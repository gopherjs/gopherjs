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
| -- flate        | yes                   |                                   |
| -- gzip         | yes                   |                                   |
| -- lzw          | yes                   |                                   |
| -- zlib         | yes                   |                                   |
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
| -- sql          | no                    | uses goroutines                   |
| -- -- driver    | yes                   |                                   |
| debug           |                       |                                   |
| -- dwarf        | not yet               |                                   |
| -- elf          | no                    |                                   |
| -- gosym        | yes                   |                                   |
| -- macho        | not yet               |                                   |
| -- pe           | yes                   |                                   |
| encoding        | (no tests)            |                                   |
| -- ascii85      | yes                   |                                   |
| -- asn1         | yes                   |                                   |
| -- base32       | yes                   |                                   |
| -- base64       | yes                   |                                   |
| -- binary       | yes                   |                                   |
| -- csv          | yes                   |                                   |
| -- gob          | no                    | uses "unsafe" heavily             |
| -- hex          | yes                   |                                   |
| -- json         | yes                   |                                   |
| -- pem          | yes                   |                                   |
| -- xml          | yes                   |                                   |
| errors          | yes                   |                                   |
| expvar          | no                    |                                   |
| flag            | not yet               |                                   |
| fmt             | yes                   |                                   |
| go              |                       |                                   |
| -- ast          | not yet               |                                   |
| -- build        | not yet               |                                   |
| -- doc          | not yet               |                                   |
| -- format       | yes                   | needs go/scanner/scanner.go patch |
| -- parser       | not yet               |                                   |
| -- printer      | yes                   |                                   |
| -- scanner      | not yet               |                                   |
| -- token        | yes                   |                                   |
| hash            | (no tests)            |                                   |
| -- adler32      | yes                   |                                   |
| -- crc32        | yes                   |                                   |
| -- crc64        | yes                   |                                   |
| -- fnv          | yes                   |                                   |
| html            | yes                   |                                   |
| -- template     | yes                   |                                   |
| image           | yes                   |                                   |
| -- color        | yes                   |                                   |
| -- -- palette   | (no tests)            |                                   |
| -- draw         | yes                   |                                   |
| -- gif          | yes                   |                                   |
| -- jpeg         | yes                   |                                   |
| -- png          | yes                   |                                   |
| index           |                       |                                   |
| -- suffixarray  | yes                   |                                   |
| io              | yes                   |                                   |
| -- ioutil       | yes                   |                                   |
| log             | not yet               |                                   |
| -- syslog       | no                    |                                   |
| math            | yes                   |                                   |
| -- big          | yes                   |                                   |
| -- cmplx        | yes                   |                                   |
| -- rand         | yes                   |                                   |
| mime            | yes                   |                                   |
| -- multipart    | no                    |                                   |
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
| -- url          | yes                   |                                   |
| os              | partially             | node.js only                      |
| -- exec         | partially             | node.js only                      |
| -- signal       | partially             | node.js only                      |
| -- user         | partially             | node.js only                      |
| path            | yes                   |                                   |
| -- filepath     | yes                   |                                   |
| reflect         | yes                   |                                   |
| regexp          | yes                   |                                   |
| -- syntax       | yes                   |                                   |
| runtime         | partially             |                                   |
| -- cgo          | no                    |                                   |
| -- debug        | no                    |                                   |
| -- pprof        | no                    |                                   |
| -- race         | no                    |                                   |
| sort            | yes                   |                                   |
| strconv         | yes                   |                                   |
| strings         | yes                   |                                   |
| sync            | partially             | stubs                             |
| -- atomic       | yes                   |                                   |
| syscall         | partially             | node.js only                      |
| testing         | yes                   |                                   |
| -- iotest       | (no tests)            |                                   |
| -- quick        | yes                   |                                   |
| text            |                       |                                   |
| -- scanner      | yes                   |                                   |
| -- tabwriter    | yes                   |                                   |
| -- template     | yes                   |                                   |
| -- -- parse     | yes                   |                                   |
| time            | partially             | AfterFunc only                    |
| unicode         | yes                   |                                   |
| -- utf16        | yes                   |                                   |
| -- utf8         | yes                   |                                   |
| unsafe          | (no tests)            |                                   |

[![Analytics](https://ga-beacon.appspot.com/UA-46799660-1/gopherjs/doc/packages.md)](https://github.com/igrigorik/ga-beacon)
