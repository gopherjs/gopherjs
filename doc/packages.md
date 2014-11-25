Supported Packages
------------------

On each commit, Circle CI automatically compiles all supported packages with GopherJS and runs their tests:

[![Circle CI](https://circleci.com/gh/gopherjs/gopherjs.svg?style=svg)](https://circleci.com/gh/gopherjs/gopherjs)

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
| -- tls          | no                    |                                   |
| -- x509         | not yet               |                                   |
| -- -- pkix      | (no tests)            |                                   |
| database        |                       |                                   |
| -- sql          | no                    |                                   |
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
| -- gob          | yes                   |                                   |
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
| -- format       | yes                   |                                   |
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
| io              | yes                   | except Pipe                       |
| -- ioutil       | yes                   |                                   |
| log             | not yet               |                                   |
| -- syslog       | no                    |                                   |
| math            | yes                   |                                   |
| -- big          | yes                   |                                   |
| -- cmplx        | yes                   |                                   |
| -- rand         | yes                   |                                   |
| mime            | yes                   |                                   |
| -- multipart    | no                    |                                   |
| net             | no                    |                                   |
| -- http         | partially             | data structures only (no net)     |
| -- -- cgi       | not yet               |                                   |
| -- -- cookiejar | yes                   |                                   |
| -- -- fcgi      | yes                   |                                   |
| -- -- httptest  | partially             |                                   |
| -- -- httputil  | partially             |                                   |
| -- -- pprof     | (no tests)            |                                   |
| -- mail         | yes                   |                                   |
| -- rpc          | not yet               |                                   |
| -- -- jsonrpc   | not yet               |                                   |
| -- smtp         | partially             | data structures only (no net)     |
| -- textproto    | yes                   |                                   |
| -- url          | yes                   |                                   |
| os              | partially             | node.js only                      |
| -- exec         | partially             | node.js only                      |
| -- signal       | partially             | node.js only                      |
| -- user         | partially             | node.js only                      |
| path            | yes                   |                                   |
| -- filepath     | yes                   |                                   |
| reflect         | yes                   | except blocking Call, Send and Recv |
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
| time            | yes                   | UTC and Local only (see [issue](https://github.com/gopherjs/gopherjs/issues/64)) |
| unicode         | yes                   |                                   |
| -- utf16        | yes                   |                                   |
| -- utf8         | yes                   |                                   |
| unsafe          | (no tests)            |                                   |
