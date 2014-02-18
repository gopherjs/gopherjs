#! /bin/bash
set -e

echo -en 'travis_fold:start:nvm\r'
echo `--- installing node.js 0.11 ---`
curl https://raw.github.com/creationix/nvm/master/install-gitless.sh | bash
. $HOME/.nvm/nvm.sh
nvm install 0.11
nvm use 0.11
echo -en 'travis_fold:end:nvm\r'

echo -en 'travis_fold:start:node-gyp\r'
echo `--- installing node-gyp ---`
npm install --global node-gyp
echo -en 'travis_fold:end:node-gyp\r'

echo -en 'travis_fold:start:syscall.node\r'
echo `--- building syscall.node module ---`
cd node-syscall
node-gyp rebuild
mkdir -p ~/.node_libraries/
cp build/Release/syscall.node ~/.node_libraries/syscall.node
echo -en 'travis_fold:end:syscall.node\r'

echo `--- running package tests ---`
$HOME/gopath/bin/gopherjs install -all
$HOME/gopath/bin/gopherjs test github.com/gopherjs/gopherjs/js
$HOME/gopath/bin/gopherjs test -short archive/tar archive/zip bufio bytes compress/bzip2 compress/flate compress/gzip compress/lzw compress/zlib container/heap container/list container/ring crypto/aes crypto/cipher crypto/des crypto/dsa crypto/ecdsa crypto/elliptic crypto/hmac crypto/md5 crypto/rand crypto/rc4 crypto/rsa crypto/sha1 crypto/sha256 crypto/sha512 crypto/subtle database/sql/driver debug/gosym debug/pe encoding/ascii85 encoding/asn1 encoding/base32 encoding/base64 encoding/binary encoding/csv encoding/hex encoding/pem encoding/xml errors fmt go/format go/printer hash/adler32 hash/crc32 hash/crc64 hash/fnv html html/template image image/color index/suffixarray io math math/cmplx math/rand mime net/url path path/filepath reflect regexp regexp/syntax sort strconv strings sync/atomic testing testing/quick text/scanner text/tabwriter text/template unicode unicode/utf8 unicode/utf16
$HOME/gopath/bin/gopherjs test -short encoding/json go/token image/draw image/gif image/jpeg image/png io/ioutil math/big text/template/parse
