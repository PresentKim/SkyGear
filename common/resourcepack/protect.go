package resourcepack

import (
	"bytes"
	"crypto/aes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Options say how a pack is prepared before it is sent to clients.
type Options struct {
	// Minify strips comments and white space from the pack's JSON and CSS
	// files.
	Minify bool
	// Encrypt encrypts the pack's files (Bedrock pack encryption, see
	// encrypt). Secret must then be set: the keys are derived from it, so
	// the same pack content always gets the same keys, and the pack version
	// follows the secret too (clients cache a pack by UUID and version).
	// Without it the pack gets an empty content.zipe, which lets the game
	// take an Ore UI page from it as from an encrypted pack.
	Encrypt bool
	Secret  []byte
	// CacheDir, if set, keeps built packs (data/packcache) keyed by content,
	// options, secret and builder version, so an unchanged pack is not built
	// again at startup. Empty means no cache.
	CacheDir string
}

// minifyJSON removes // and /* */ comments (JSON UI files have them) and
// white space from a JSON document, leaving strings and the order of keys
// untouched. It returns an error if what is left is not valid JSON.
func minifyJSON(src []byte) ([]byte, error) {
	var out bytes.Buffer
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '"':
			j := i + 1
			for j < len(src) && src[j] != '"' {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			if j >= len(src) {
				return nil, errors.New("unterminated string")
			}
			out.Write(src[i : j+1])
			i = j + 1
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := bytes.Index(src[i+2:], []byte("*/"))
			if end < 0 {
				return nil, errors.New("unterminated comment")
			}
			i += end + 4
		default:
			out.WriteByte(c)
			i++
		}
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, bytes.TrimPrefix(out.Bytes(), []byte("\xef\xbb\xbf"))); err != nil {
		return nil, err
	}
	return compact.Bytes(), nil
}

// minifyDir minifies every .json file under dir but the manifest, and every
// .css file (the Ore UI page's style sheets). A JSON file that is not valid
// JSON once its comments are gone is an error (the game would not read it
// either).
func minifyDir(dir string) error {
	return filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		var minify func([]byte) ([]byte, error)
		switch {
		case strings.HasSuffix(p, ".json") && filepath.Base(p) != "manifest.json":
			minify = minifyJSON
		case strings.HasSuffix(p, ".css"):
			minify = minifyCSS
		default:
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		min, err := minify(b)
		if err != nil {
			rel, _ := filepath.Rel(dir, p)
			return fmt.Errorf("minify %s: %w", rel, err)
		}
		return os.WriteFile(p, min, 0o644)
	})
}

// minifyCSS removes /* */ comments from a style sheet and folds its white
// space: a run becomes one space, and none is kept next to { } ; , or at the
// ends. Strings and url() are copied as they are. A space before : is kept
// (".a :hover" is not ".a:hover"), so no selector changes its meaning.
func minifyCSS(src []byte) ([]byte, error) {
	var out bytes.Buffer
	space := false
	tight := func(c byte) bool { return c == '{' || c == '}' || c == ';' || c == ',' }
	flush := func(next byte) {
		if space && out.Len() > 0 && !tight(out.Bytes()[out.Len()-1]) && !tight(next) {
			out.WriteByte(' ')
		}
		space = false
	}
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := bytes.Index(src[i+2:], []byte("*/"))
			if end < 0 {
				return nil, errors.New("unterminated comment")
			}
			i += end + 4
			space = true
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f':
			space = true
			i++
		case c == '"' || c == '\'':
			j := i + 1
			for j < len(src) && src[j] != c && src[j] != '\n' {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			if j >= len(src) || src[j] != c {
				return nil, errors.New("unterminated string")
			}
			flush(c)
			out.Write(src[i : j+1])
			i = j + 1
		case c == '(' && bytes.HasSuffix(out.Bytes(), []byte("url")) && !space:
			end := bytes.IndexByte(src[i:], ')')
			if end < 0 {
				return nil, errors.New("unterminated url(")
			}
			out.Write(src[i : i+end+1])
			i += end + 1
		default:
			flush(c)
			if b := out.Bytes(); c == '}' && len(b) > 0 && b[len(b)-1] == ';' {
				out.Truncate(len(b) - 1) // a declaration list needs no last ;
			}
			out.WriteByte(c)
			i++
		}
	}
	return out.Bytes(), nil
}

// LoadSecret reads the encryption secret at path, creating it from random
// bytes the first time. Keep the file: a new secret changes every pack's keys
// and version, so clients download the packs again.
func LoadSecret(path string) ([]byte, error) {
	if b, err := os.ReadFile(path); err == nil && len(bytes.TrimSpace(b)) >= 32 {
		return bytes.TrimSpace(b), nil
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	secret := []byte(hex.EncodeToString(raw))
	return secret, os.WriteFile(path, append(secret, '\n'), 0o600)
}

const keyChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// deriveKey makes a 32 character key (AES-256) from the secret and a label.
func deriveKey(secret []byte, label string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(label))
	sum := mac.Sum(nil)
	key := make([]byte, 32)
	for i := range key {
		key[i] = keyChars[int(sum[i])%len(keyChars)]
	}
	return key
}

// cfb8 encrypts (or with decrypt set, decrypts) data in place with AES in
// CFB8 mode, the IV being the first 16 bytes of the 32 byte key (the mode
// Bedrock uses for pack encryption; the Go library has only CFB128).
func cfb8(data, key []byte, decrypt bool) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	// The shift register is a window over buf: block i is buf[i:i+16] and
	// the cipher byte of step i lands at buf[i+16]. After each chunk the last
	// 16 bytes move to the front, so no byte is copied per step.
	const chunk = 4096
	buf := make([]byte, 16+min(chunk, len(data)))
	copy(buf, key[:16])
	var tmp [16]byte
	for off := 0; off < len(data); off += chunk {
		part := data[off:min(off+chunk, len(data))]
		for j := range part {
			block.Encrypt(tmp[:], buf[j:j+16])
			in := part[j]
			part[j] ^= tmp[0]
			if decrypt {
				buf[16+j] = in
			} else {
				buf[16+j] = part[j]
			}
		}
		copy(buf, buf[len(part):len(part)+16])
	}
	return nil
}

type contentEntry struct {
	Path string `json:"path"`
	Key  string `json:"key,omitempty"`
}

// encryptDir encrypts the files of the pack at dir the way Bedrock expects
// (what marketplace packs use): every file but manifest.json and
// pack_icon.png is encrypted with a key of its own, and contents.json lists
// the paths with their keys, itself encrypted with the pack's content key
// (which the server hands to clients). The content key is returned.
func encryptDir(dir, packUUID string, secret []byte) (string, error) {
	var (
		entries   []contentEntry
		files     []string
		encrypted []int
	)
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if rel != "manifest.json" && rel != "pack_icon.png" {
			encrypted = append(encrypted, len(entries))
		}
		entries = append(entries, contentEntry{Path: rel})
		files = append(files, p)
		return nil
	})
	if err != nil {
		return "", err
	}
	if err := encryptFiles(entries, files, encrypted, secret); err != nil {
		return "", err
	}
	list, err := json.Marshal(struct {
		Version int            `json:"version"`
		Content []contentEntry `json:"content"`
	}{1, entries})
	if err != nil {
		return "", err
	}
	master := deriveKey(secret, "master\x00"+packUUID)
	if err := cfb8(list, master, false); err != nil {
		return "", err
	}
	// The header: version 0, the format's magic number, padding, the UUID of
	// the pack as a length-prefixed string, zeros up to 256 bytes.
	var head bytes.Buffer
	head.Write(make([]byte, 4))
	head.WriteString("\xfc\xb9\xcf\x9b")
	head.Write(make([]byte, 8))
	head.WriteByte(byte(len(packUUID)))
	head.WriteString(packUUID)
	head.Write(make([]byte, 256-head.Len()))
	head.Write(list)
	if err := os.WriteFile(filepath.Join(dir, "contents.json"), head.Bytes(), 0o644); err != nil {
		return "", err
	}
	return string(master), nil
}

// encryptFiles encrypts files[i] in place for every i of encrypted, at most
// GOMAXPROCS at once, and sets entries[i].Key. It returns the error of the
// first such file in walk order.
func encryptFiles(entries []contentEntry, files []string, encrypted []int, secret []byte) error {
	errs := make([]error, len(entries))
	next := make(chan int)
	var wg sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), len(encrypted)) {
		wg.Go(func() {
			for i := range next {
				entries[i].Key, errs[i] = encryptFile(files[i], entries[i].Path, secret)
			}
		})
	}
	for _, i := range encrypted {
		next <- i
	}
	close(next)
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// encryptFile encrypts the file at p, rel in the pack, with its own key and
// returns the key.
func encryptFile(p, rel string, secret []byte) (string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	key := deriveKey(secret, "file\x00"+rel)
	if err := cfb8(b, key, false); err != nil {
		return "", err
	}
	return string(key), os.WriteFile(p, b, 0o644)
}
