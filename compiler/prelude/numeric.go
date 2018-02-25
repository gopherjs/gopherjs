package prelude

const numericMinified = `var $min=Math.min,$mod=function(r,n){return r%n},$parseInt=parseInt,$parseFloat=function(r){return null!=r&&r.constructor===Number?r:parseFloat(r)},$froundBuf=new Float32Array(1),$fround=Math.fround||function(r){return $froundBuf[0]=r,$froundBuf[0]},$imul=Math.imul||function(r,n){var t=65535&r,$=65535&n;return t*$+((r>>>16&65535)*$+t*(n>>>16&65535)<<16>>>0)>>0},$floatKey=function(r){return r!=r?($idCounter++,"NaN$"+$idCounter):String(r)},$flatten64=function(r){return 4294967296*r.$high+r.$low},$shiftLeft64=function(r,n){return 0===n?r:n<32?new r.constructor(r.$high<<n|r.$low>>>32-n,r.$low<<n>>>0):n<64?new r.constructor(r.$low<<n-32,0):new r.constructor(0,0)},$shiftRightInt64=function(r,n){return 0===n?r:n<32?new r.constructor(r.$high>>n,(r.$low>>>n|r.$high<<32-n)>>>0):n<64?new r.constructor(r.$high>>31,r.$high>>n-32>>>0):r.$high<0?new r.constructor(-1,4294967295):new r.constructor(0,0)},$shiftRightUint64=function(r,n){return 0===n?r:n<32?new r.constructor(r.$high>>>n,(r.$low>>>n|r.$high<<32-n)>>>0):n<64?new r.constructor(0,r.$high>>>n-32):new r.constructor(0,0)},$mul64=function(r,n){var t=0,$=0;0!=(1&n.$low)&&(t=r.$high,$=r.$low);for(var o=1;o<32;o++)0!=(n.$low&1<<o)&&(t+=r.$high<<o|r.$low>>>32-o,$+=r.$low<<o>>>0);for(o=0;o<32;o++)0!=(n.$high&1<<o)&&(t+=r.$low<<o);return new r.constructor(t,$)},$div64=function(r,n,t){0===n.$high&&0===n.$low&&$throwRuntimeError("integer divide by zero");var $=1,o=1,e=r.$high,i=r.$low;e<0&&($=-1,o=-1,e=-e,0!==i&&(e--,i=4294967296-i));var a=n.$high,u=n.$low;n.$high<0&&($*=-1,a=-a,0!==u&&(a--,u=4294967296-u));for(var c=0,h=0,l=0;a<2147483648&&(e>a||e===a&&i>u);)a=(a<<1|u>>>31)>>>0,u=u<<1>>>0,l++;for(var g=0;g<=l;g++)c=c<<1|h>>>31,h=h<<1>>>0,(e>a||e===a&&i>=u)&&(e-=a,(i-=u)<0&&(e--,i+=4294967296),4294967296===++h&&(c++,h=0)),u=(u>>>1|a<<31)>>>0,a>>>=1;return t?new r.constructor(e*o,i*o):new r.constructor(c*$,h*$)},$divComplex=function(r,n){var t=r.$real===1/0||r.$real===-1/0||r.$imag===1/0||r.$imag===-1/0,$=n.$real===1/0||n.$real===-1/0||n.$imag===1/0||n.$imag===-1/0,o=!t&&(r.$real!=r.$real||r.$imag!=r.$imag),e=!$&&(n.$real!=n.$real||n.$imag!=n.$imag);if(o||e)return new r.constructor(NaN,NaN);if(t&&!$)return new r.constructor(1/0,1/0);if(!t&&$)return new r.constructor(0,0);if(0===n.$real&&0===n.$imag)return 0===r.$real&&0===r.$imag?new r.constructor(NaN,NaN):new r.constructor(1/0,1/0);if(Math.abs(n.$real)<=Math.abs(n.$imag)){var i=n.$real/n.$imag,a=n.$real*i+n.$imag;return new r.constructor((r.$real*i+r.$imag)/a,(r.$imag*i-r.$real)/a)}i=n.$imag/n.$real,a=n.$imag*i+n.$real;return new r.constructor((r.$imag*i+r.$real)/a,(r.$imag-r.$real*i)/a)};`
const numeric = `
var $min = Math.min;
var $mod = function(x, y) { return x % y; };
var $parseInt = parseInt;
var $parseFloat = function(f) {
  if (f !== undefined && f !== null && f.constructor === Number) {
    return f;
  }
  return parseFloat(f);
};

var $froundBuf = new Float32Array(1);
var $fround = Math.fround || function(f) {
  $froundBuf[0] = f;
  return $froundBuf[0];
};

var $imul = Math.imul || function(a, b) {
  var ah = (a >>> 16) & 0xffff;
  var al = a & 0xffff;
  var bh = (b >>> 16) & 0xffff;
  var bl = b & 0xffff;
  return ((al * bl) + (((ah * bl + al * bh) << 16) >>> 0) >> 0);
};

var $floatKey = function(f) {
  if (f !== f) {
    $idCounter++;
    return "NaN$" + $idCounter;
  }
  return String(f);
};

var $flatten64 = function(x) {
  return x.$high * 4294967296 + x.$low;
};

var $shiftLeft64 = function(x, y) {
  if (y === 0) {
    return x;
  }
  if (y < 32) {
    return new x.constructor(x.$high << y | x.$low >>> (32 - y), (x.$low << y) >>> 0);
  }
  if (y < 64) {
    return new x.constructor(x.$low << (y - 32), 0);
  }
  return new x.constructor(0, 0);
};

var $shiftRightInt64 = function(x, y) {
  if (y === 0) {
    return x;
  }
  if (y < 32) {
    return new x.constructor(x.$high >> y, (x.$low >>> y | x.$high << (32 - y)) >>> 0);
  }
  if (y < 64) {
    return new x.constructor(x.$high >> 31, (x.$high >> (y - 32)) >>> 0);
  }
  if (x.$high < 0) {
    return new x.constructor(-1, 4294967295);
  }
  return new x.constructor(0, 0);
};

var $shiftRightUint64 = function(x, y) {
  if (y === 0) {
    return x;
  }
  if (y < 32) {
    return new x.constructor(x.$high >>> y, (x.$low >>> y | x.$high << (32 - y)) >>> 0);
  }
  if (y < 64) {
    return new x.constructor(0, x.$high >>> (y - 32));
  }
  return new x.constructor(0, 0);
};

var $mul64 = function(x, y) {
  var high = 0, low = 0;
  if ((y.$low & 1) !== 0) {
    high = x.$high;
    low = x.$low;
  }
  for (var i = 1; i < 32; i++) {
    if ((y.$low & 1<<i) !== 0) {
      high += x.$high << i | x.$low >>> (32 - i);
      low += (x.$low << i) >>> 0;
    }
  }
  for (var i = 0; i < 32; i++) {
    if ((y.$high & 1<<i) !== 0) {
      high += x.$low << i;
    }
  }
  return new x.constructor(high, low);
};

var $div64 = function(x, y, returnRemainder) {
  if (y.$high === 0 && y.$low === 0) {
    $throwRuntimeError("integer divide by zero");
  }

  var s = 1;
  var rs = 1;

  var xHigh = x.$high;
  var xLow = x.$low;
  if (xHigh < 0) {
    s = -1;
    rs = -1;
    xHigh = -xHigh;
    if (xLow !== 0) {
      xHigh--;
      xLow = 4294967296 - xLow;
    }
  }

  var yHigh = y.$high;
  var yLow = y.$low;
  if (y.$high < 0) {
    s *= -1;
    yHigh = -yHigh;
    if (yLow !== 0) {
      yHigh--;
      yLow = 4294967296 - yLow;
    }
  }

  var high = 0, low = 0, n = 0;
  while (yHigh < 2147483648 && ((xHigh > yHigh) || (xHigh === yHigh && xLow > yLow))) {
    yHigh = (yHigh << 1 | yLow >>> 31) >>> 0;
    yLow = (yLow << 1) >>> 0;
    n++;
  }
  for (var i = 0; i <= n; i++) {
    high = high << 1 | low >>> 31;
    low = (low << 1) >>> 0;
    if ((xHigh > yHigh) || (xHigh === yHigh && xLow >= yLow)) {
      xHigh = xHigh - yHigh;
      xLow = xLow - yLow;
      if (xLow < 0) {
        xHigh--;
        xLow += 4294967296;
      }
      low++;
      if (low === 4294967296) {
        high++;
        low = 0;
      }
    }
    yLow = (yLow >>> 1 | yHigh << (32 - 1)) >>> 0;
    yHigh = yHigh >>> 1;
  }

  if (returnRemainder) {
    return new x.constructor(xHigh * rs, xLow * rs);
  }
  return new x.constructor(high * s, low * s);
};

var $divComplex = function(n, d) {
  var ninf = n.$real === Infinity || n.$real === -Infinity || n.$imag === Infinity || n.$imag === -Infinity;
  var dinf = d.$real === Infinity || d.$real === -Infinity || d.$imag === Infinity || d.$imag === -Infinity;
  var nnan = !ninf && (n.$real !== n.$real || n.$imag !== n.$imag);
  var dnan = !dinf && (d.$real !== d.$real || d.$imag !== d.$imag);
  if(nnan || dnan) {
    return new n.constructor(NaN, NaN);
  }
  if (ninf && !dinf) {
    return new n.constructor(Infinity, Infinity);
  }
  if (!ninf && dinf) {
    return new n.constructor(0, 0);
  }
  if (d.$real === 0 && d.$imag === 0) {
    if (n.$real === 0 && n.$imag === 0) {
      return new n.constructor(NaN, NaN);
    }
    return new n.constructor(Infinity, Infinity);
  }
  var a = Math.abs(d.$real);
  var b = Math.abs(d.$imag);
  if (a <= b) {
    var ratio = d.$real / d.$imag;
    var denom = d.$real * ratio + d.$imag;
    return new n.constructor((n.$real * ratio + n.$imag) / denom, (n.$imag * ratio - n.$real) / denom);
  }
  var ratio = d.$imag / d.$real;
  var denom = d.$imag * ratio + d.$real;
  return new n.constructor((n.$imag * ratio + n.$real) / denom, (n.$imag - n.$real * ratio) / denom);
};
`
