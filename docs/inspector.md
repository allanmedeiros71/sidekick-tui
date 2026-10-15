# Inspector

Open it from the menu, or directly with `sidekick inspect`. Paste or type
some text and the inspector works out what it is, decodes it, and shows the
text's encodings and hashes below. `↑`/`↓` (or `tab`) select a line, `enter`
copies its value to the clipboard and `esc` goes back to the menu. Long
values are cut to fit the popup, but `enter` always copies the whole value.

For scripts and quick checks, pass the input on the command line or pipe it
in. A single trailing line break is dropped from piped input, so
`echo hello | sidekick inspect` hashes `hello`, not `hello\n`.

```sh
sidekick inspect 'aGVsbG8gd29ybGQ='
kubectl get secret app -o jsonpath='{.data.token}' | sidekick inspect
curl -s https://api.example.com/items | sidekick inspect
```

## What it recognizes

The input is matched against these kinds, in order, and the first match is
shown at the top:

| Kind             | Example                                | Shows |
| ---------------- | -------------------------------------- | ----- |
| JWT              | `eyJhbGci...`, also `Bearer eyJ...`    | algorithm, whether it is valid now (from `exp` and `nbf`), `iat`/`nbf`/`exp` as dates, `iss`/`sub`/`aud`, header and payload as pretty JSON. The signature is **not** verified. |
| UUID             | `f47ac10b-58cc-...`, `{...}`, `urn:uuid:...` | version and variant; the embedded time for versions 1, 6 and 7 and the node for 1 and 6; upper case, hex and base64 forms |
| JSON             | `{"a": 1}`, `[1, 2]`, `"a\tb"`         | key or item count, pretty-printed and minified (key order is kept); a JSON string is shown unescaped. Invalid JSON shows the error and its byte offset. |
| URL              | `postgres://app:pw@db:5432/app?sslmode=require` | scheme, user, host, port, decoded path, each query parameter and the fragment. The password is masked. |
| Percent-encoded  | `a%2Fb+c`                              | decoded, and also form-decoded (`+` as a space) when the text has a `+` |
| Hex              | `48656c6c6f`, `0x4869`, `de:ad:be:ef`  | byte count, the text when it is readable, the base64 form, and which hash a digest of that length could be (MD5, SHA-1, SHA-256...) |
| Base64           | `aGVsbG8=`, unpadded or URL-safe       | which variant it is, the decoded text (or hex for binary data) and, when the decoded text is JSON, the JSON |

To avoid noise, plain numbers are not taken for hex (the calculator is the
place for those), and short words that happen to be valid hex or base64
(`cafe`, `test`) are treated as plain text unless they decode to readable
text.

## Encodings and hashes

Every input, recognized or not, is also shown:

- **encode**: length in bytes, characters and lines; base64, unpadded
  base64url, URL query encoding, hex and as a JSON string.
- **hashes**: MD5, SHA-1, SHA-256, SHA-512 and CRC-32, in hex.

## UUIDs

Type `uuid` to get a fresh random version 4 UUID, a time-ordered version 7
UUID and an upper-case version 4, ready to copy.

## Clipboard

Copying uses `pbcopy` on macOS and `wl-copy`, `xclip` or `xsel` on Linux.
When none of them works, as over SSH, sidekick sends an OSC 52 escape
sequence instead, which most terminals turn into a clipboard write. Inside
tmux this needs `set -g set-clipboard on`.
