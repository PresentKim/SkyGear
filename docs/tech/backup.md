# 백업

`platform.toml` 의 `[Backup]` 이 켜져 있으면(기본) 서버가 스스로 백업한다. 파일은 `Dir`(기본 `data/backups`)에 쌓인다.

| 대상 | 시점 | 파일 | 보관 |
|---|---|---|---|
| 플랫폼 DB(`platform.db`) | `DatabaseMinutes`(기본 60)마다, 서버를 끌 때 한 번 더 | `db-<날짜-시각>.sqlite` | 최근 `DatabaseKeep`(기본 48)개 |
| 세이브 월드 폴더 | 인스턴스가 닫힐 때(호스트가 나가거나 서버가 꺼질 때). 월드가 닫힌 뒤라 파일이 일관된다 | `world-<세이브 ID>-<날짜-시각>.tar.gz` | 세이브마다 최근 `WorldKeep`(기본 5)개 |

- DB 는 `VACUUM INTO` 로 쓰는 중에도 일관된 사본을 만든다(WAL 파일이 따로 필요 없다).
- 활성 인스턴스의 월드는 열려 있는 동안 복사하지 않는다(LevelDB 파일이 바뀌는 중이라 일관되지 않다). 월드는 1분마다(`WorldSaveSeconds`) 디스크에 쓰이므로 충돌해도 최대 그 정도만 되돌아간다.
- 같은 디스크에 두면 디스크 고장에는 소용이 없다. `Dir` 을 다른 디스크로 두거나 `data/backups` 를 주기적으로 다른 곳으로 복사한다.

## 되돌리기

서버를 끄고 한다.
- DB: 고른 `db-….sqlite` 를 `data/platform.db` 로 복사하고 같은 폴더의 `platform.db-wal`, `platform.db-shm` 을 지운다.
- 세이브 월드: `data/saves/<세이브 ID>/world` 를 지우고 그 자리에 `world-<ID>-….tar.gz` 를 푼다(`mkdir world && tar -xzf <파일> -C world`).
