# 리소스팩 처리(압축·암호화)

`internal/resourcepack` 가 서버를 켤 때 `assets/resourcepack/*` 의 팩을 읽어 클라이언트에 보낼 모양으로 만든다. 소스 파일은 그대로 두고 임시 폴더에 복사한 것만 바꾼다.

1. **버전**: 내용 해시로 `manifest.json` 버전을 자동으로 정한다(손대지 않는다). 암호화를 켜면 비밀값도 해시에 들어간다.
2. **JSON minify**(`[ResourcePack] Minify`, 기본 켬): `.json` 파일의 `//`·`/* */` 주석과 공백을 지운다(키 순서·문자열은 그대로). `manifest.json` 은 건드리지 않는다. 주석을 지운 뒤에도 올바른 JSON 이 아니면 서버가 켜지지 않는다(그런 파일은 게임도 못 읽는다). JSON UI 파일에 주석이 많아 용량이 꽤 준다.
3. **암호화**(`[ResourcePack] Encrypt`, 기본 켬): 바닐라 마켓 팩과 같은 방식이다.
   - 파일마다 32글자 키로 AES-256-CFB8(IV = 키 앞 16바이트)로 암호화한다. `manifest.json`·`pack_icon.png` 는 그대로 둔다.
   - `contents.json` 에 경로와 키를 적고(앞 256바이트는 머리: 버전 0, 매직 `FC B9 CF 9B`, 팩 UUID), 목록은 팩의 **콘텐츠 키**로 같은 방식으로 암호화한다. 콘텐츠 키는 접속할 때 서버가 클라이언트에 준다(gophertunnel `Pack.WithContentKey`).
   - 키는 비밀값에서 HMAC 으로 만든다(같은 내용 = 같은 키·같은 버전, 클라이언트 캐시가 유지된다). 비밀값은 `data/resourcepack.secret`(처음 켤 때 만들어짐, 권한 0600). **이 파일을 잃거나 바꾸면 모든 팩의 키·버전이 바뀌어 클라이언트가 팩을 다시 받는다.** 저장소에 넣지 않는다(`/data/` 는 무시됨).
   - 형식은 커뮤니티 도구(`akmalfairuz/bedrockpack`, `EncryptMyPack`)와 같다. 테스트(`protect_test.go`)가 우리 쪽에서 되풀이로 풀어 보는 것까지 확인한다. **실제 클라이언트에서 열리는지는 실기로 확인해야 한다**(안 되면 `platform.toml` 에서 `Encrypt = false`).

## 한계

암호화는 파일을 줍는 것을 번거롭게 할 뿐이다. 키를 클라이언트에 주므로 접속하는 클라이언트는 풀 수 있고, 도구(`bedrockpack steal`)로 한 번에 풀린다. 실질적인 보호가 아니라 억제다. 보이면 안 되는 것은 팩에 두지 않는다.

## 설정(`platform.toml`)

```toml
[ResourcePack]
Minify = true
Encrypt = true
```
