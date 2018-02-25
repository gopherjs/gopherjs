package prelude

const jsmappingMinified = `var $jsObjectPtr,$jsErrorPtr,$needsExternalization=function(e){switch(e.kind){case $kindBool:case $kindInt:case $kindInt8:case $kindInt16:case $kindInt32:case $kindUint:case $kindUint8:case $kindUint16:case $kindUint32:case $kindUintptr:case $kindFloat32:case $kindFloat64:return!1;default:return e!==$jsObjectPtr}},$externalize=function(e,r){if(r===$jsObjectPtr)return e;switch(r.kind){case $kindBool:case $kindInt:case $kindInt8:case $kindInt16:case $kindInt32:case $kindUint:case $kindUint8:case $kindUint16:case $kindUint32:case $kindUintptr:case $kindFloat32:case $kindFloat64:return e;case $kindInt64:case $kindUint64:return $flatten64(e);case $kindArray:return $needsExternalization(r.elem)?$mapArray(e,function(e){return $externalize(e,r.elem)}):e;case $kindFunc:return $externalizeFunction(e,r,!1);case $kindInterface:return e===$ifaceNil?null:e.constructor===$jsObjectPtr?e.$val.object:$externalize(e.$val,e.constructor);case $kindMap:for(var n={},t=$keys(e),a=0;a<t.length;a++){var i=e[t[a]];n[$externalize(i.k,r.key)]=$externalize(i.v,r.elem)}return n;case $kindPtr:return e===r.nil?null:$externalize(e.$get(),r.elem);case $kindSlice:return $needsExternalization(r.elem)?$mapArray($sliceToArray(e),function(e){return $externalize(e,r.elem)}):$sliceToArray(e);case $kindString:if($isASCII(e))return e;var s,c="";for(a=0;a<e.length;a+=s[1]){var $=(s=$decodeRune(e,a))[0];if($>65535){var l=Math.floor(($-65536)/1024)+55296,u=($-65536)%1024+56320;c+=String.fromCharCode(l,u)}else c+=String.fromCharCode($)}return c;case $kindStruct:var o=$packages.time;if(void 0!==o&&e.constructor===o.Time.ptr){var d=$div64(e.UnixNano(),new $Int64(0,1e6));return new Date($flatten64(d))}var k={},f=function(e,r){if(r===$jsObjectPtr)return e;switch(r.kind){case $kindPtr:return e===r.nil?k:f(e.$get(),r.elem);case $kindStruct:var n=r.fields[0];return f(e[n.prop],n.typ);case $kindInterface:return f(e.$val,e.constructor);default:return k}},p=f(e,r);if(p!==k)return p;p={};for(a=0;a<r.fields.length;a++){var m=r.fields[a];m.exported&&(p[m.name]=$externalize(e[m.prop],m.typ))}return p}$throwRuntimeError("cannot externalize "+r.string)},$externalizeFunction=function(e,r,n){return e===$throwNilPointerError?null:(void 0===e.$externalizeWrapper&&($checkForDeadlock=!1,e.$externalizeWrapper=function(){for(var t=[],a=0;a<r.params.length;a++){if(r.variadic&&a===r.params.length-1){for(var i=r.params[a].elem,s=[],c=a;c<arguments.length;c++)s.push($internalize(arguments[c],i));t.push(new r.params[a](s));break}t.push($internalize(arguments[a],r.params[a]))}var $=e.apply(n?this:void 0,t);switch(r.results.length){case 0:return;case 1:return $externalize($,r.results[0]);default:for(a=0;a<r.results.length;a++)$[a]=$externalize($[a],r.results[a]);return $}}),e.$externalizeWrapper)},$internalize=function(e,r,n){if(r===$jsObjectPtr)return e;if(r===$jsObjectPtr.elem&&$throwRuntimeError("cannot internalize js.Object, use *js.Object instead"),e&&void 0!==e.__internal_object__)return $assertType(e.__internal_object__,r,!1);var t=$packages.time;if(void 0!==t&&r===t.Time)return null!=e&&e.constructor===Date||$throwRuntimeError("cannot internalize time.Time from "+typeof e+", must be Date"),t.Unix(new $Int64(0,0),new $Int64(0,1e6*e.getTime()));switch(r.kind){case $kindBool:return!!e;case $kindInt:return parseInt(e);case $kindInt8:return parseInt(e)<<24>>24;case $kindInt16:return parseInt(e)<<16>>16;case $kindInt32:return parseInt(e)>>0;case $kindUint:return parseInt(e);case $kindUint8:return parseInt(e)<<24>>>24;case $kindUint16:return parseInt(e)<<16>>>16;case $kindUint32:case $kindUintptr:return parseInt(e)>>>0;case $kindInt64:case $kindUint64:return new r(0,e);case $kindFloat32:case $kindFloat64:return parseFloat(e);case $kindArray:return e.length!==r.len&&$throwRuntimeError("got array with wrong size from JavaScript native"),$mapArray(e,function(e){return $internalize(e,r.elem)});case $kindFunc:return function(){for(var t=[],a=0;a<r.params.length;a++){if(r.variadic&&a===r.params.length-1){for(var i=r.params[a].elem,s=arguments[a],c=0;c<s.$length;c++)t.push($externalize(s.$array[s.$offset+c],i));break}t.push($externalize(arguments[a],r.params[a]))}var $=e.apply(n,t);switch(r.results.length){case 0:return;case 1:return $internalize($,r.results[0]);default:for(a=0;a<r.results.length;a++)$[a]=$internalize($[a],r.results[a]);return $}};case $kindInterface:if(0!==r.methods.length&&$throwRuntimeError("cannot internalize "+r.string),null===e)return $ifaceNil;if(void 0===e)return new $jsObjectPtr(void 0);switch(e.constructor){case Int8Array:return new($sliceType($Int8))(e);case Int16Array:return new($sliceType($Int16))(e);case Int32Array:return new($sliceType($Int))(e);case Uint8Array:return new($sliceType($Uint8))(e);case Uint16Array:return new($sliceType($Uint16))(e);case Uint32Array:return new($sliceType($Uint))(e);case Float32Array:return new($sliceType($Float32))(e);case Float64Array:return new($sliceType($Float64))(e);case Array:return $internalize(e,$sliceType($emptyInterface));case Boolean:return new $Bool(!!e);case Date:return void 0===t?new $jsObjectPtr(e):new t.Time($internalize(e,t.Time));case Function:var a=$funcType([$sliceType($emptyInterface)],[$jsObjectPtr],!0);return new a($internalize(e,a));case Number:return new $Float64(parseFloat(e));case String:return new $String($internalize(e,$String));default:if($global.Node&&e instanceof $global.Node)return new $jsObjectPtr(e);var i=$mapType($String,$emptyInterface);return new i($internalize(e,i))}case $kindMap:for(var s={},c=$keys(e),$=0;$<c.length;$++){var l=$internalize(c[$],r.key);s[r.key.keyFor(l)]={k:l,v:$internalize(e[c[$]],r.elem)}}return s;case $kindPtr:if(r.elem.kind===$kindStruct)return $internalize(e,r.elem);case $kindSlice:return new r($mapArray(e,function(e){return $internalize(e,r.elem)}));case $kindString:if(e=String(e),$isASCII(e))return e;var u="";for($=0;$<e.length;){var o=e.charCodeAt($);if(55296<=o&&o<=56319){var d=e.charCodeAt($+1);u+=$encodeRune(1024*(o-55296)+d-56320+65536),$+=2}else u+=$encodeRune(o),$++}return u;case $kindStruct:var k={},f=function(r){if(r===$jsObjectPtr)return e;switch(r===$jsObjectPtr.elem&&$throwRuntimeError("cannot internalize js.Object, use *js.Object instead"),r.kind){case $kindPtr:return f(r.elem);case $kindStruct:var n=r.fields[0],t=f(n.typ);if(t!==k){var a=new r.ptr;return a[n.prop]=t,a}return k;default:return k}},p=f(r);if(p!==k)return p}$throwRuntimeError("cannot internalize "+r.string)},$isASCII=function(e){for(var r=0;r<e.length;r++)if(e.charCodeAt(r)>=128)return!1;return!0};`
const jsmapping = `
var $jsObjectPtr, $jsErrorPtr;

var $needsExternalization = function(t) {
  switch (t.kind) {
    case $kindBool:
    case $kindInt:
    case $kindInt8:
    case $kindInt16:
    case $kindInt32:
    case $kindUint:
    case $kindUint8:
    case $kindUint16:
    case $kindUint32:
    case $kindUintptr:
    case $kindFloat32:
    case $kindFloat64:
      return false;
    default:
      return t !== $jsObjectPtr;
  }
};

var $externalize = function(v, t) {
  if (t === $jsObjectPtr) {
    return v;
  }
  switch (t.kind) {
  case $kindBool:
  case $kindInt:
  case $kindInt8:
  case $kindInt16:
  case $kindInt32:
  case $kindUint:
  case $kindUint8:
  case $kindUint16:
  case $kindUint32:
  case $kindUintptr:
  case $kindFloat32:
  case $kindFloat64:
    return v;
  case $kindInt64:
  case $kindUint64:
    return $flatten64(v);
  case $kindArray:
    if ($needsExternalization(t.elem)) {
      return $mapArray(v, function(e) { return $externalize(e, t.elem); });
    }
    return v;
  case $kindFunc:
    return $externalizeFunction(v, t, false);
  case $kindInterface:
    if (v === $ifaceNil) {
      return null;
    }
    if (v.constructor === $jsObjectPtr) {
      return v.$val.object;
    }
    return $externalize(v.$val, v.constructor);
  case $kindMap:
    var m = {};
    var keys = $keys(v);
    for (var i = 0; i < keys.length; i++) {
      var entry = v[keys[i]];
      m[$externalize(entry.k, t.key)] = $externalize(entry.v, t.elem);
    }
    return m;
  case $kindPtr:
    if (v === t.nil) {
      return null;
    }
    return $externalize(v.$get(), t.elem);
  case $kindSlice:
    if ($needsExternalization(t.elem)) {
      return $mapArray($sliceToArray(v), function(e) { return $externalize(e, t.elem); });
    }
    return $sliceToArray(v);
  case $kindString:
    if ($isASCII(v)) {
      return v;
    }
    var s = "", r;
    for (var i = 0; i < v.length; i += r[1]) {
      r = $decodeRune(v, i);
      var c = r[0];
      if (c > 0xFFFF) {
        var h = Math.floor((c - 0x10000) / 0x400) + 0xD800;
        var l = (c - 0x10000) % 0x400 + 0xDC00;
        s += String.fromCharCode(h, l);
        continue;
      }
      s += String.fromCharCode(c);
    }
    return s;
  case $kindStruct:
    var timePkg = $packages["time"];
    if (timePkg !== undefined && v.constructor === timePkg.Time.ptr) {
      var milli = $div64(v.UnixNano(), new $Int64(0, 1000000));
      return new Date($flatten64(milli));
    }

    var noJsObject = {};
    var searchJsObject = function(v, t) {
      if (t === $jsObjectPtr) {
        return v;
      }
      switch (t.kind) {
      case $kindPtr:
        if (v === t.nil) {
          return noJsObject;
        }
        return searchJsObject(v.$get(), t.elem);
      case $kindStruct:
        var f = t.fields[0];
        return searchJsObject(v[f.prop], f.typ);
      case $kindInterface:
        return searchJsObject(v.$val, v.constructor);
      default:
        return noJsObject;
      }
    };
    var o = searchJsObject(v, t);
    if (o !== noJsObject) {
      return o;
    }

    o = {};
    for (var i = 0; i < t.fields.length; i++) {
      var f = t.fields[i];
      if (!f.exported) {
        continue;
      }
      o[f.name] = $externalize(v[f.prop], f.typ);
    }
    return o;
  }
  $throwRuntimeError("cannot externalize " + t.string);
};

var $externalizeFunction = function(v, t, passThis) {
  if (v === $throwNilPointerError) {
    return null;
  }
  if (v.$externalizeWrapper === undefined) {
    $checkForDeadlock = false;
    v.$externalizeWrapper = function() {
      var args = [];
      for (var i = 0; i < t.params.length; i++) {
        if (t.variadic && i === t.params.length - 1) {
          var vt = t.params[i].elem, varargs = [];
          for (var j = i; j < arguments.length; j++) {
            varargs.push($internalize(arguments[j], vt));
          }
          args.push(new (t.params[i])(varargs));
          break;
        }
        args.push($internalize(arguments[i], t.params[i]));
      }
      var result = v.apply(passThis ? this : undefined, args);
      switch (t.results.length) {
      case 0:
        return;
      case 1:
        return $externalize(result, t.results[0]);
      default:
        for (var i = 0; i < t.results.length; i++) {
          result[i] = $externalize(result[i], t.results[i]);
        }
        return result;
      }
    };
  }
  return v.$externalizeWrapper;
};

var $internalize = function(v, t, recv) {
  if (t === $jsObjectPtr) {
    return v;
  }
  if (t === $jsObjectPtr.elem) {
    $throwRuntimeError("cannot internalize js.Object, use *js.Object instead");
  }
  if (v && v.__internal_object__ !== undefined) {
    return $assertType(v.__internal_object__, t, false);
  }
  var timePkg = $packages["time"];
  if (timePkg !== undefined && t === timePkg.Time) {
    if (!(v !== null && v !== undefined && v.constructor === Date)) {
      $throwRuntimeError("cannot internalize time.Time from " + typeof v + ", must be Date");
    }
    return timePkg.Unix(new $Int64(0, 0), new $Int64(0, v.getTime() * 1000000));
  }
  switch (t.kind) {
  case $kindBool:
    return !!v;
  case $kindInt:
    return parseInt(v);
  case $kindInt8:
    return parseInt(v) << 24 >> 24;
  case $kindInt16:
    return parseInt(v) << 16 >> 16;
  case $kindInt32:
    return parseInt(v) >> 0;
  case $kindUint:
    return parseInt(v);
  case $kindUint8:
    return parseInt(v) << 24 >>> 24;
  case $kindUint16:
    return parseInt(v) << 16 >>> 16;
  case $kindUint32:
  case $kindUintptr:
    return parseInt(v) >>> 0;
  case $kindInt64:
  case $kindUint64:
    return new t(0, v);
  case $kindFloat32:
  case $kindFloat64:
    return parseFloat(v);
  case $kindArray:
    if (v.length !== t.len) {
      $throwRuntimeError("got array with wrong size from JavaScript native");
    }
    return $mapArray(v, function(e) { return $internalize(e, t.elem); });
  case $kindFunc:
    return function() {
      var args = [];
      for (var i = 0; i < t.params.length; i++) {
        if (t.variadic && i === t.params.length - 1) {
          var vt = t.params[i].elem, varargs = arguments[i];
          for (var j = 0; j < varargs.$length; j++) {
            args.push($externalize(varargs.$array[varargs.$offset + j], vt));
          }
          break;
        }
        args.push($externalize(arguments[i], t.params[i]));
      }
      var result = v.apply(recv, args);
      switch (t.results.length) {
      case 0:
        return;
      case 1:
        return $internalize(result, t.results[0]);
      default:
        for (var i = 0; i < t.results.length; i++) {
          result[i] = $internalize(result[i], t.results[i]);
        }
        return result;
      }
    };
  case $kindInterface:
    if (t.methods.length !== 0) {
      $throwRuntimeError("cannot internalize " + t.string);
    }
    if (v === null) {
      return $ifaceNil;
    }
    if (v === undefined) {
      return new $jsObjectPtr(undefined);
    }
    switch (v.constructor) {
    case Int8Array:
      return new ($sliceType($Int8))(v);
    case Int16Array:
      return new ($sliceType($Int16))(v);
    case Int32Array:
      return new ($sliceType($Int))(v);
    case Uint8Array:
      return new ($sliceType($Uint8))(v);
    case Uint16Array:
      return new ($sliceType($Uint16))(v);
    case Uint32Array:
      return new ($sliceType($Uint))(v);
    case Float32Array:
      return new ($sliceType($Float32))(v);
    case Float64Array:
      return new ($sliceType($Float64))(v);
    case Array:
      return $internalize(v, $sliceType($emptyInterface));
    case Boolean:
      return new $Bool(!!v);
    case Date:
      if (timePkg === undefined) {
        /* time package is not present, internalize as &js.Object{Date} so it can be externalized into original Date. */
        return new $jsObjectPtr(v);
      }
      return new timePkg.Time($internalize(v, timePkg.Time));
    case Function:
      var funcType = $funcType([$sliceType($emptyInterface)], [$jsObjectPtr], true);
      return new funcType($internalize(v, funcType));
    case Number:
      return new $Float64(parseFloat(v));
    case String:
      return new $String($internalize(v, $String));
    default:
      if ($global.Node && v instanceof $global.Node) {
        return new $jsObjectPtr(v);
      }
      var mapType = $mapType($String, $emptyInterface);
      return new mapType($internalize(v, mapType));
    }
  case $kindMap:
    var m = {};
    var keys = $keys(v);
    for (var i = 0; i < keys.length; i++) {
      var k = $internalize(keys[i], t.key);
      m[t.key.keyFor(k)] = { k: k, v: $internalize(v[keys[i]], t.elem) };
    }
    return m;
  case $kindPtr:
    if (t.elem.kind === $kindStruct) {
      return $internalize(v, t.elem);
    }
  case $kindSlice:
    return new t($mapArray(v, function(e) { return $internalize(e, t.elem); }));
  case $kindString:
    v = String(v);
    if ($isASCII(v)) {
      return v;
    }
    var s = "";
    var i = 0;
    while (i < v.length) {
      var h = v.charCodeAt(i);
      if (0xD800 <= h && h <= 0xDBFF) {
        var l = v.charCodeAt(i + 1);
        var c = (h - 0xD800) * 0x400 + l - 0xDC00 + 0x10000;
        s += $encodeRune(c);
        i += 2;
        continue;
      }
      s += $encodeRune(h);
      i++;
    }
    return s;
  case $kindStruct:
    var noJsObject = {};
    var searchJsObject = function(t) {
      if (t === $jsObjectPtr) {
        return v;
      }
      if (t === $jsObjectPtr.elem) {
        $throwRuntimeError("cannot internalize js.Object, use *js.Object instead");
      }
      switch (t.kind) {
      case $kindPtr:
        return searchJsObject(t.elem);
      case $kindStruct:
        var f = t.fields[0];
        var o = searchJsObject(f.typ);
        if (o !== noJsObject) {
          var n = new t.ptr();
          n[f.prop] = o;
          return n;
        }
        return noJsObject;
      default:
        return noJsObject;
      }
    };
    var o = searchJsObject(t);
    if (o !== noJsObject) {
      return o;
    }
  }
  $throwRuntimeError("cannot internalize " + t.string);
};

/* $isASCII reports whether string s contains only ASCII characters. */
var $isASCII = function(s) {
  for (var i = 0; i < s.length; i++) {
    if (s.charCodeAt(i) >= 128) {
      return false;
    }
  }
  return true;
};
`
