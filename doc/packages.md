| Name            | Supported (Tests OK?) | Comment                           |
| --------------- | --------------------- | --------------------------------- |
| archive         |                       |                                   |
| -- tar          | yes                   |                                   |
| -- zip          | yes                   |                                   |
| bufio           | yes                   |                                   |
| builtin         | (no tests)            |                                   |
| bytes           | yes                   |                                   |
| compress        |                       |                                   |
| -- bzip2        | yes                   |                                   |
| -- flate        | maybe                 | tests need goroutines             |
| -- gzip         | yes                   |                                   |
| -- lzw          | maybe                 | tests need goroutines             |
| -- zlib         | maybe                 | tests need goroutines             |
| container       |                       |                                   |
| -- heap         | yes                   |                                   |
| -- list         | yes                   |                                   |
| -- ring         | yes                   |                                   |
| crypto          | (no tests)            |                                   |
| -- aes          | yes                   |                                   |
| -- cipher       | yes                   |                                   |
| -- des          | yes                   |                                   |
| -- dsa          | yes                   |                                   |
| -- ecdsa        | yes                   |                                   |
| -- elliptic     | yes                   |                                   |
| -- hmac         | yes                   |                                   |
| -- md5          | yes                   |                                   |
| -- rand         | yes                   |                                   |
| -- rc4          | yes                   |                                   |
| -- rsa          | yes                   |                                   |
| -- sha1         | yes                   |                                   |
| -- sha256       | yes                   |                                   |
| -- sha512       | yes                   |                                   |
| -- subtle       | yes                   |                                   |
| -- tls          | no                    | needs C code                      |
| -- x509         | no                    | needs C code                      |
| -- -- pkix      | no                    | needs C code                      |
| database        |                       |                                   |
| -- sql          | not yet               |                                   |
| -- -- driver    | not yet               |                                   |
| debug           |                       |                                   |
| -- dwarf        | not yet               |                                   |
| -- elf          | not yet               |                                   |
| -- gosym        | yes                   |                                   |
| -- macho        | not yet               |                                   |
| -- pe           | not yet               |                                   |
| encoding        | (no tests)            |                                   |
| -- ascii85      | yes                   |                                   |
| -- asn1         | not yet               | needs reflection                  |
| -- base32       | yes                   |                                   |
| -- base64       | yes                   |                                   |
| -- binary       | not yet               |                                   |
| -- csv          | yes                   |                                   |
| -- gob          | not yet               | needs reflection                  |
| -- hex          | yes                   |                                   |
| -- json         | not yet               | needs reflection                  |
| -- pem          | yes                   |                                   |
| -- xml          | not yet               | needs reflection                  |
| errors          | yes                   |                                   |
| expvar          | not yet               |                                   |
| flag            | not yet               |                                   |
| fmt             | not yet               |                                   |
| go              |                       |                                   |
| -- ast          | not yet               |                                   |
| -- build        | not yet               |                                   |
| -- doc          | not yet               |                                   |
| -- format       | yes                   | needs go/scanner/scanner.go patch |
| -- parser       | not yet               |                                   |
| -- printer      | not yet               |                                   |
| -- scanner      | not yet               |                                   |
| -- token        | not yet               |                                   |
| hash            | (no tests)            |                                   |
| -- adler32      | yes                   |                                   |
| -- crc32        | yes                   |                                   |
| -- crc64        | yes                   |                                   |
| -- fnv          | yes                   |                                   |
| html            | yes                   |                                   |
| -- template     | not yet               |                                   |
| image           | yes                   |                                   |
| -- color        | yes                   |                                   |
| -- -- palette   | not yet               |                                   |
| -- draw         | yes                   |                                   |
| -- gif          | yes                   |                                   |
| -- jpeg         | yes                   |                                   |
| -- png          | maybe                 | tests need goroutines             |
| index           |                       |                                   |
| -- suffixarray  | not yet               |                                   |
| io              | yes                   |                                   |
| -- ioutil       | yes                   |                                   |
| log             | not yet               |                                   |
| -- syslog       | not yet               |                                   |
| math            | yes                   |                                   |
| -- big          | yes                   |                                   |
| -- cmplx        | yes                   |                                   |
| -- rand         | yes                   |                                   |
| mime            | yes                   |                                   |
| -- multipart    | not yet               |                                   |
| net             | no                    | needs C code                      |
| -- http         | no                    |                                   |
| -- -- cgi       | no                    |                                   |
| -- -- cookiejar | no                    |                                   |
| -- -- fcgi      | no                    |                                   |
| -- -- httptest  | no                    |                                   |
| -- -- httputil  | no                    |                                   |
| -- -- pprof     | no                    |                                   |
| -- mail         | no                    |                                   |
| -- rpc          | no                    |                                   |
| -- -- jsonrpc   | no                    |                                   |
| -- smtp         | no                    |                                   |
| -- textproto    | no                    |                                   |
| -- url          | not yet               |                                   |
| os              | partially             | node.js only                      |
| -- exec         | partially             | node.js only                      |
| -- signal       | partially             | node.js only                      |
| -- user         | partially             | node.js only                      |
| path            | yes                   |                                   |
| -- filepath     | not yet               |                                   |
| reflect         | partially             |                                   |
| regexp          | yes                   |                                   |
| -- syntax       | not yet               |                                   |
| runtime         | partially             |                                   |
| -- cgo          | not yet               |                                   |
| -- debug        | not yet               |                                   |
| -- pprof        | not yet               |                                   |
| -- race         | not yet               |                                   |
| sort            | yes                   |                                   |
| strconv         | yes                   |                                   |
| strings         | yes                   |                                   |
| sync            | partially             | stubs                             |
| -- atomic       | partially             | stubs                             |
| syscall         | partially             | node.js only                      |
| testing         | yes                   |                                   |
| -- iotest       | not yet               |                                   |
| -- quick        | yes                   |                                   |
| text            |                       |                                   |
| -- scanner      | yes                   |                                   |
| -- tabwriter    | yes                   |                                   |
| -- template     | not yet               |                                   |
| -- -- parse     | not yet               |                                   |
| time            | partially             | AfterFunc only                    |
| unicode         | yes                   |                                   |
| -- utf16        | yes                   |                                   |
| -- utf8         | yes                   |                                   |
| unsafe          | (no tests)            |                                   |