package prelude

const Prelude = prelude + numeric + types + goroutines + jsmapping

// Minified is Prelude ran through UglifyJS 3 via https://skalman.github.io/UglifyJS-online/
// with default options.
const Minified = `var $global,$module;if(Error.stackTraceLimit=1/0,"undefined"!=typeof window?$global=window:"undefined"!=typeof self?$global=self:"undefined"!=typeof global?($global=global).require=require:$global=this,void 0===$global||void 0===$global.Array)throw new Error("no global object found");"undefined"!=typeof module&&($module=module);var $throwRuntimeError,$packages={},$idCounter=0,$keys=function(r){return r?Object.keys(r):[]},$flushConsole=function(){},$throwNilPointerError=function(){$throwRuntimeError("invalid memory address or nil pointer dereference")},$call=function(r,e,t){return r.apply(e,t)},$makeFunc=function(r){return function(){return $externalize(r(this,new($sliceType($jsObjectPtr))($global.Array.prototype.slice.call(arguments,[]))),$emptyInterface)}},$unused=function(r){},$mapArray=function(r,e){for(var t=new r.constructor(r.length),n=0;n<r.length;n++)t[n]=e(r[n]);return t},$methodVal=function(r,e){var t=r.$methodVals||{};r.$methodVals=t;var n=t[e];if(void 0!==n)return n;var o=r[e];return n=function(){$stackDepthOffset--;try{return o.apply(r,arguments)}finally{$stackDepthOffset++}},t[e]=n,n},$methodExpr=function(r,e){var t=r.prototype[e];return void 0===t.$expr&&(t.$expr=function(){$stackDepthOffset--;try{return r.wrapped&&(arguments[0]=new r(arguments[0])),Function.call.apply(t,arguments)}finally{$stackDepthOffset++}}),t.$expr},$ifaceMethodExprs={},$ifaceMethodExpr=function(r){var e=$ifaceMethodExprs["$"+r];return void 0===e&&(e=$ifaceMethodExprs["$"+r]=function(){$stackDepthOffset--;try{return Function.call.apply(arguments[0][r],arguments)}finally{$stackDepthOffset++}}),e},$subslice=function(r,e,t,n){void 0===t&&(t=r.$length),void 0===n&&(n=r.$capacity),(e<0||t<e||n<t||t>r.$capacity||n>r.$capacity)&&$throwRuntimeError("slice bounds out of range");var o=new r.constructor(r.$array);return o.$offset=r.$offset+e,o.$length=t-e,o.$capacity=n-e,o},$substring=function(r,e,t){return(e<0||t<e||t>r.length)&&$throwRuntimeError("slice bounds out of range"),r.substring(e,t)},$sliceToArray=function(r){return r.$array.constructor!==Array?r.$array.subarray(r.$offset,r.$offset+r.$length):r.$array.slice(r.$offset,r.$offset+r.$length)},$decodeRune=function(r,e){var t=r.charCodeAt(e);if(t<128)return[t,1];if(t!=t||t<192)return[65533,1];var n=r.charCodeAt(e+1);if(n!=n||n<128||192<=n)return[65533,1];if(t<224)return(a=(31&t)<<6|63&n)<=127?[65533,1]:[a,2];var o=r.charCodeAt(e+2);if(o!=o||o<128||192<=o)return[65533,1];if(t<240)return(a=(15&t)<<12|(63&n)<<6|63&o)<=2047?[65533,1]:55296<=a&&a<=57343?[65533,1]:[a,3];var a,i=r.charCodeAt(e+3);return i!=i||i<128||192<=i?[65533,1]:t<248?(a=(7&t)<<18|(63&n)<<12|(63&o)<<6|63&i)<=65535||1114111<a?[65533,1]:[a,4]:[65533,1]},$encodeRune=function(r){return(r<0||r>1114111||55296<=r&&r<=57343)&&(r=65533),r<=127?String.fromCharCode(r):r<=2047?String.fromCharCode(192|r>>6,128|63&r):r<=65535?String.fromCharCode(224|r>>12,128|r>>6&63,128|63&r):String.fromCharCode(240|r>>18,128|r>>12&63,128|r>>6&63,128|63&r)},$stringToBytes=function(r){for(var e=new Uint8Array(r.length),t=0;t<r.length;t++)e[t]=r.charCodeAt(t);return e},$bytesToString=function(r){if(0===r.$length)return"";for(var e="",t=0;t<r.$length;t+=1e4)e+=String.fromCharCode.apply(void 0,r.$array.subarray(r.$offset+t,r.$offset+Math.min(r.$length,t+1e4)));return e},$stringToRunes=function(r){for(var e,t=new Int32Array(r.length),n=0,o=0;o<r.length;o+=e[1],n++)e=$decodeRune(r,o),t[n]=e[0];return t.subarray(0,n)},$runesToString=function(r){if(0===r.$length)return"";for(var e="",t=0;t<r.$length;t++)e+=$encodeRune(r.$array[r.$offset+t]);return e},$copyString=function(r,e){for(var t=Math.min(e.length,r.$length),n=0;n<t;n++)r.$array[r.$offset+n]=e.charCodeAt(n);return t},$copySlice=function(r,e){var t=Math.min(e.$length,r.$length);return $copyArray(r.$array,e.$array,r.$offset,e.$offset,t,r.constructor.elem),t},$copyArray=function(r,e,t,n,o,a){if(0!==o&&(r!==e||t!==n))if(e.subarray)r.set(e.subarray(n,n+o),t);else{switch(a.kind){case $kindArray:case $kindStruct:if(r===e&&t>n){for(var i=o-1;i>=0;i--)a.copy(r[t+i],e[n+i]);return}for(i=0;i<o;i++)a.copy(r[t+i],e[n+i]);return}if(r===e&&t>n)for(i=o-1;i>=0;i--)r[t+i]=e[n+i];else for(i=0;i<o;i++)r[t+i]=e[n+i]}},$clone=function(r,e){var t=e.zero();return e.copy(t,r),t},$pointerOfStructConversion=function(r,e){void 0===r.$proxies&&(r.$proxies={},r.$proxies[r.constructor.string]=r);var t=r.$proxies[e.string];if(void 0===t){for(var n={},o=0;o<e.elem.fields.length;o++)!function(e){n[e]={get:function(){return r[e]},set:function(t){r[e]=t}}}(e.elem.fields[o].prop);(t=Object.create(e.prototype,n)).$val=t,r.$proxies[e.string]=t,t.$proxies=r.$proxies}return t},$append=function(r){return $internalAppend(r,arguments,1,arguments.length-1)},$appendSlice=function(r,e){if(e.constructor===String){var t=$stringToBytes(e);return $internalAppend(r,t,0,t.length)}return $internalAppend(r,e.$array,e.$offset,e.$length)},$internalAppend=function(r,e,t,n){if(0===n)return r;var o=r.$array,a=r.$offset,i=r.$length+n,$=r.$capacity;if(i>$)if(a=0,$=Math.max(i,r.$capacity<1024?2*r.$capacity:Math.floor(5*r.$capacity/4)),r.$array.constructor===Array){(o=r.$array.slice(r.$offset,r.$offset+r.$length)).length=$;for(var c=r.constructor.elem.zero,f=r.$length;f<$;f++)o[f]=c()}else(o=new r.$array.constructor($)).set(r.$array.subarray(r.$offset,r.$offset+r.$length));$copyArray(o,e,a+r.$length,t,n,r.constructor.elem);var u=new r.constructor(o);return u.$offset=a,u.$length=i,u.$capacity=$,u},$equal=function(r,e,t){if(t===$jsObjectPtr)return r===e;switch(t.kind){case $kindComplex64:case $kindComplex128:return r.$real===e.$real&&r.$imag===e.$imag;case $kindInt64:case $kindUint64:return r.$high===e.$high&&r.$low===e.$low;case $kindArray:if(r.length!==e.length)return!1;for(var n=0;n<r.length;n++)if(!$equal(r[n],e[n],t.elem))return!1;return!0;case $kindStruct:for(n=0;n<t.fields.length;n++){var o=t.fields[n];if(!$equal(r[o.prop],e[o.prop],o.typ))return!1}return!0;case $kindInterface:return $interfaceIsEqual(r,e);default:return r===e}},$interfaceIsEqual=function(r,e){return r===$ifaceNil||e===$ifaceNil?r===e:r.constructor===e.constructor&&(r.constructor===$jsObjectPtr?r.object===e.object:(r.constructor.comparable||$throwRuntimeError("comparing uncomparable type "+r.constructor.string),$equal(r.$val,e.$val,r.constructor)))};`

const prelude = `Error.stackTraceLimit = Infinity;

var $global, $module;
if (typeof window !== "undefined") { /* web page */
  $global = window;
} else if (typeof self !== "undefined") { /* web worker */
  $global = self;
} else if (typeof global !== "undefined") { /* Node.js */
  $global = global;
  $global.require = require;
} else { /* others (e.g. Nashorn) */
  $global = this;
}

if ($global === undefined || $global.Array === undefined) {
  throw new Error("no global object found");
}
if (typeof module !== "undefined") {
  $module = module;
}

var $packages = {}, $idCounter = 0;
var $keys = function(m) { return m ? Object.keys(m) : []; };
var $flushConsole = function() {};
var $throwRuntimeError; /* set by package "runtime" */
var $throwNilPointerError = function() { $throwRuntimeError("invalid memory address or nil pointer dereference"); };
var $call = function(fn, rcvr, args) { return fn.apply(rcvr, args); };
var $makeFunc = function(fn) { return function() { return $externalize(fn(this, new ($sliceType($jsObjectPtr))($global.Array.prototype.slice.call(arguments, []))), $emptyInterface); }; };
var $unused = function(v) {};

var $mapArray = function(array, f) {
  var newArray = new array.constructor(array.length);
  for (var i = 0; i < array.length; i++) {
    newArray[i] = f(array[i]);
  }
  return newArray;
};

var $methodVal = function(recv, name) {
  var vals = recv.$methodVals || {};
  recv.$methodVals = vals; /* noop for primitives */
  var f = vals[name];
  if (f !== undefined) {
    return f;
  }
  var method = recv[name];
  f = function() {
    $stackDepthOffset--;
    try {
      return method.apply(recv, arguments);
    } finally {
      $stackDepthOffset++;
    }
  };
  vals[name] = f;
  return f;
};

var $methodExpr = function(typ, name) {
  var method = typ.prototype[name];
  if (method.$expr === undefined) {
    method.$expr = function() {
      $stackDepthOffset--;
      try {
        if (typ.wrapped) {
          arguments[0] = new typ(arguments[0]);
        }
        return Function.call.apply(method, arguments);
      } finally {
        $stackDepthOffset++;
      }
    };
  }
  return method.$expr;
};

var $ifaceMethodExprs = {};
var $ifaceMethodExpr = function(name) {
  var expr = $ifaceMethodExprs["$" + name];
  if (expr === undefined) {
    expr = $ifaceMethodExprs["$" + name] = function() {
      $stackDepthOffset--;
      try {
        return Function.call.apply(arguments[0][name], arguments);
      } finally {
        $stackDepthOffset++;
      }
    };
  }
  return expr;
};

var $subslice = function(slice, low, high, max) {
  if (high === undefined) {
    high = slice.$length;
  }
  if (max === undefined) {
    max = slice.$capacity
  }
  if (low < 0 || high < low || max < high || high > slice.$capacity || max > slice.$capacity) {
    $throwRuntimeError("slice bounds out of range");
  }
  var s = new slice.constructor(slice.$array);
  s.$offset = slice.$offset + low;
  s.$length = high - low;
  s.$capacity = max - low;
  return s;
};

var $substring = function(str, low, high) {
  if (low < 0 || high < low || high > str.length) {
    $throwRuntimeError("slice bounds out of range");
  }
  return str.substring(low, high);
};

var $sliceToArray = function(slice) {
  if (slice.$array.constructor !== Array) {
    return slice.$array.subarray(slice.$offset, slice.$offset + slice.$length);
  }
  return slice.$array.slice(slice.$offset, slice.$offset + slice.$length);
};

var $decodeRune = function(str, pos) {
  var c0 = str.charCodeAt(pos);

  if (c0 < 0x80) {
    return [c0, 1];
  }

  if (c0 !== c0 || c0 < 0xC0) {
    return [0xFFFD, 1];
  }

  var c1 = str.charCodeAt(pos + 1);
  if (c1 !== c1 || c1 < 0x80 || 0xC0 <= c1) {
    return [0xFFFD, 1];
  }

  if (c0 < 0xE0) {
    var r = (c0 & 0x1F) << 6 | (c1 & 0x3F);
    if (r <= 0x7F) {
      return [0xFFFD, 1];
    }
    return [r, 2];
  }

  var c2 = str.charCodeAt(pos + 2);
  if (c2 !== c2 || c2 < 0x80 || 0xC0 <= c2) {
    return [0xFFFD, 1];
  }

  if (c0 < 0xF0) {
    var r = (c0 & 0x0F) << 12 | (c1 & 0x3F) << 6 | (c2 & 0x3F);
    if (r <= 0x7FF) {
      return [0xFFFD, 1];
    }
    if (0xD800 <= r && r <= 0xDFFF) {
      return [0xFFFD, 1];
    }
    return [r, 3];
  }

  var c3 = str.charCodeAt(pos + 3);
  if (c3 !== c3 || c3 < 0x80 || 0xC0 <= c3) {
    return [0xFFFD, 1];
  }

  if (c0 < 0xF8) {
    var r = (c0 & 0x07) << 18 | (c1 & 0x3F) << 12 | (c2 & 0x3F) << 6 | (c3 & 0x3F);
    if (r <= 0xFFFF || 0x10FFFF < r) {
      return [0xFFFD, 1];
    }
    return [r, 4];
  }

  return [0xFFFD, 1];
};

var $encodeRune = function(r) {
  if (r < 0 || r > 0x10FFFF || (0xD800 <= r && r <= 0xDFFF)) {
    r = 0xFFFD;
  }
  if (r <= 0x7F) {
    return String.fromCharCode(r);
  }
  if (r <= 0x7FF) {
    return String.fromCharCode(0xC0 | r >> 6, 0x80 | (r & 0x3F));
  }
  if (r <= 0xFFFF) {
    return String.fromCharCode(0xE0 | r >> 12, 0x80 | (r >> 6 & 0x3F), 0x80 | (r & 0x3F));
  }
  return String.fromCharCode(0xF0 | r >> 18, 0x80 | (r >> 12 & 0x3F), 0x80 | (r >> 6 & 0x3F), 0x80 | (r & 0x3F));
};

var $stringToBytes = function(str) {
  var array = new Uint8Array(str.length);
  for (var i = 0; i < str.length; i++) {
    array[i] = str.charCodeAt(i);
  }
  return array;
};

var $bytesToString = function(slice) {
  if (slice.$length === 0) {
    return "";
  }
  var str = "";
  for (var i = 0; i < slice.$length; i += 10000) {
    str += String.fromCharCode.apply(undefined, slice.$array.subarray(slice.$offset + i, slice.$offset + Math.min(slice.$length, i + 10000)));
  }
  return str;
};

var $stringToRunes = function(str) {
  var array = new Int32Array(str.length);
  var rune, j = 0;
  for (var i = 0; i < str.length; i += rune[1], j++) {
    rune = $decodeRune(str, i);
    array[j] = rune[0];
  }
  return array.subarray(0, j);
};

var $runesToString = function(slice) {
  if (slice.$length === 0) {
    return "";
  }
  var str = "";
  for (var i = 0; i < slice.$length; i++) {
    str += $encodeRune(slice.$array[slice.$offset + i]);
  }
  return str;
};

var $copyString = function(dst, src) {
  var n = Math.min(src.length, dst.$length);
  for (var i = 0; i < n; i++) {
    dst.$array[dst.$offset + i] = src.charCodeAt(i);
  }
  return n;
};

var $copySlice = function(dst, src) {
  var n = Math.min(src.$length, dst.$length);
  $copyArray(dst.$array, src.$array, dst.$offset, src.$offset, n, dst.constructor.elem);
  return n;
};

var $copyArray = function(dst, src, dstOffset, srcOffset, n, elem) {
  if (n === 0 || (dst === src && dstOffset === srcOffset)) {
    return;
  }

  if (src.subarray) {
    dst.set(src.subarray(srcOffset, srcOffset + n), dstOffset);
    return;
  }

  switch (elem.kind) {
  case $kindArray:
  case $kindStruct:
    if (dst === src && dstOffset > srcOffset) {
      for (var i = n - 1; i >= 0; i--) {
        elem.copy(dst[dstOffset + i], src[srcOffset + i]);
      }
      return;
    }
    for (var i = 0; i < n; i++) {
      elem.copy(dst[dstOffset + i], src[srcOffset + i]);
    }
    return;
  }

  if (dst === src && dstOffset > srcOffset) {
    for (var i = n - 1; i >= 0; i--) {
      dst[dstOffset + i] = src[srcOffset + i];
    }
    return;
  }
  for (var i = 0; i < n; i++) {
    dst[dstOffset + i] = src[srcOffset + i];
  }
};

var $clone = function(src, type) {
  var clone = type.zero();
  type.copy(clone, src);
  return clone;
};

var $pointerOfStructConversion = function(obj, type) {
  if(obj.$proxies === undefined) {
    obj.$proxies = {};
    obj.$proxies[obj.constructor.string] = obj;
  }
  var proxy = obj.$proxies[type.string];
  if (proxy === undefined) {
    var properties = {};
    for (var i = 0; i < type.elem.fields.length; i++) {
      (function(fieldProp) {
        properties[fieldProp] = {
          get: function() { return obj[fieldProp]; },
          set: function(value) { obj[fieldProp] = value; }
        };
      })(type.elem.fields[i].prop);
    }
    proxy = Object.create(type.prototype, properties);
    proxy.$val = proxy;
    obj.$proxies[type.string] = proxy;
    proxy.$proxies = obj.$proxies;
  }
  return proxy;
};

var $append = function(slice) {
  return $internalAppend(slice, arguments, 1, arguments.length - 1);
};

var $appendSlice = function(slice, toAppend) {
  if (toAppend.constructor === String) {
    var bytes = $stringToBytes(toAppend);
    return $internalAppend(slice, bytes, 0, bytes.length);
  }
  return $internalAppend(slice, toAppend.$array, toAppend.$offset, toAppend.$length);
};

var $internalAppend = function(slice, array, offset, length) {
  if (length === 0) {
    return slice;
  }

  var newArray = slice.$array;
  var newOffset = slice.$offset;
  var newLength = slice.$length + length;
  var newCapacity = slice.$capacity;

  if (newLength > newCapacity) {
    newOffset = 0;
    newCapacity = Math.max(newLength, slice.$capacity < 1024 ? slice.$capacity * 2 : Math.floor(slice.$capacity * 5 / 4));

    if (slice.$array.constructor === Array) {
      newArray = slice.$array.slice(slice.$offset, slice.$offset + slice.$length);
      newArray.length = newCapacity;
      var zero = slice.constructor.elem.zero;
      for (var i = slice.$length; i < newCapacity; i++) {
        newArray[i] = zero();
      }
    } else {
      newArray = new slice.$array.constructor(newCapacity);
      newArray.set(slice.$array.subarray(slice.$offset, slice.$offset + slice.$length));
    }
  }

  $copyArray(newArray, array, newOffset + slice.$length, offset, length, slice.constructor.elem);

  var newSlice = new slice.constructor(newArray);
  newSlice.$offset = newOffset;
  newSlice.$length = newLength;
  newSlice.$capacity = newCapacity;
  return newSlice;
};

var $equal = function(a, b, type) {
  if (type === $jsObjectPtr) {
    return a === b;
  }
  switch (type.kind) {
  case $kindComplex64:
  case $kindComplex128:
    return a.$real === b.$real && a.$imag === b.$imag;
  case $kindInt64:
  case $kindUint64:
    return a.$high === b.$high && a.$low === b.$low;
  case $kindArray:
    if (a.length !== b.length) {
      return false;
    }
    for (var i = 0; i < a.length; i++) {
      if (!$equal(a[i], b[i], type.elem)) {
        return false;
      }
    }
    return true;
  case $kindStruct:
    for (var i = 0; i < type.fields.length; i++) {
      var f = type.fields[i];
      if (!$equal(a[f.prop], b[f.prop], f.typ)) {
        return false;
      }
    }
    return true;
  case $kindInterface:
    return $interfaceIsEqual(a, b);
  default:
    return a === b;
  }
};

var $interfaceIsEqual = function(a, b) {
  if (a === $ifaceNil || b === $ifaceNil) {
    return a === b;
  }
  if (a.constructor !== b.constructor) {
    return false;
  }
  if (a.constructor === $jsObjectPtr) {
    return a.object === b.object;
  }
  if (!a.constructor.comparable) {
    $throwRuntimeError("comparing uncomparable type " + a.constructor.string);
  }
  return $equal(a.$val, b.$val, a.constructor);
};
`
