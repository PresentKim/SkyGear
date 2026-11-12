# 클라이언트 청크 캐시 (blob cache)

배드락 클라이언트는 서버가 보낸 서브청크·바이옴 데이터를 xxhash64 값으로 저장해 두는 **blob 캐시**가 있다.
캐시는 접속이 끝나도 남아 있어서, 다시 접속하거나 같은 청크를 다시 받을 때 바뀌지 않은 데이터는 해시만 받으면 된다.
Dragonfly(`third_party/dragonfly/server/session/chunk.go`)는 이 방식을 이미 구현해 두었고, 클라이언트가 캐시를 켜고 접속하면 그대로 쓴다.

## 패킷 흐름

| 순서 | 패킷 | 방향 | 내용 |
|---|---|---|---|
| 1 | `ClientCacheStatus{Enabled}` | 클라이언트 → 서버 | 접속할 때 한 번 보낸다. 캐시를 지원하는지 알린다(닌텐도 스위치 등은 `false`). gophertunnel 의 `Conn.ClientCacheEnabled()` 가 이 값이다 |
| 2 | `LevelChunk{CacheEnabled: true, BlobHashes}` | 서버 → 클라이언트 | 서브청크 요청 모드에서는 바이옴 blob 해시 하나만 보낸다(`SubChunkCount 0`, `SubChunkLimit`) |
| 3 | `SubChunkRequest` | 클라이언트 → 서버 | 서브청크를 요청한다 |
| 4 | `SubChunk{CacheEnabled: true}` | 서버 → 클라이언트 | 각 항목의 `BlobHash` 에 블록 데이터 해시를, `RawPayload` 에는 블록 엔티티 NBT 만 담는다. 공기뿐인 서브청크는 `SuccessAllAir` 로 보낸다 |
| 5 | `ClientCacheBlobStatus{HitHashes, MissHashes}` | 클라이언트 → 서버 | 이미 있는 해시(hit)와 없는 해시(miss) |
| 6 | `ClientCacheMissResponse{Blobs}` | 서버 → 클라이언트 | miss 의 실제 데이터. 서버는 보낸 blob 을 hit·miss 응답이 올 때까지 세션에 들고 있다(최대 4096개) |

캐시는 해시만 보고 내용이 같다고 판단한다. 서브청크 blob 에는 블록 런타임 ID 가 들어가므로, 커스텀 블록 목록이 바뀌면 같은 블록이라도 데이터와 해시가 함께 바뀐다. 그래서 예전 캐시의 blob 이 잘못 쓰이지는 않는다.

## 시험 도구 (`test/chunk-caching` 브랜치)

- `common/sghook/chunkcache.go`: 연결마다 캐시 패킷 수를 센다. `sghook.ChunkCache(p)`, `ResetChunkCache(p)`, `ResendChunks(p, tx)`(로드된 청크를 모두 다시 보냄), `ChunkCacheLog`(blob 상태·miss 응답 INFO 로그).
- `/sgdebug chunkcache <stats|reset|resend|log>`: 운영자용 명령. 실제 클라이언트로 `reset` → `resend` → `stats` 순서로 쓰면 다시 보낸 청크가 모두 hit 인지 본다.
- `tools/cachebot`: 캐시를 켠 가짜 클라이언트. blob 을 파일에 저장해 두어 재접속을 흉내 낸다. 서버의 `AuthEnabled = false`, 봇 이름을 `Admin.Operators` 에 넣고 쓴다.

```sh
go run ./tools/cachebot -resend -wait 10s   # 첫 접속 + 재전송
go run ./tools/cachebot -wait 10s           # 재접속 (cachebot.blobs 유지)
```

로비(서브청크 대부분이 공기)에서 잰 결과:

| 경우 | LevelChunk | blob 서브청크 | hit | miss | miss 응답 |
|---|---|---|---|---|---|
| 첫 접속(빈 캐시) | 861 | 8 | 829 | 40 | 10 blob, 9030 B |
| 같은 접속에서 재전송 | 861 | 8 | 869 | 0 | 0 |
| 재접속(캐시 유지) | 861 | 8 | 868 | 1 | 1 blob, 1040 B |

miss 40 건에 blob 이 10 개인 것은 같은 바이옴 해시가 여러 청크에서 겹치기 때문이다(서버는 해시마다 한 번만 보낸다).

### 알게 된 점

- 클라이언트가 한 배치에 패킷을 너무 많이 보내면 gophertunnel 이 연결을 닫는다(`maximumInBatch`). 처음 만든 봇은 재전송 받은 861 청크의 `SubChunkRequest` 를 한 번에 보내서 로그 없이 끊겼다. 그래서 봇은 틱마다 64 개씩만 보낸다.
- Dragonfly 세션은 이런 읽기 오류로 끊길 때 로그를 남기지 않는다(`handlePackets` 가 그냥 끝난다).
