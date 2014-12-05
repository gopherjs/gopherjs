package prelude

const jsmapping = `
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
    case $kindInterface:
      return t !== $packages["github.com/gopherjs/gopherjs/js"].Object;
    default:
      return true;
  }
};

var $externalize = function(v, t) {
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
    if (v === $throwNilPointerError) {
      return null;
    }
    if (v.$externalizeWrapper === undefined) {
      $checkForDeadlock = false;
      var convert = false;
      var i;
      for (i = 0; i < t.params.length; i++) {
        convert = convert || (t.params[i] !== $packages["github.com/gopherjs/gopherjs/js"].Object);
      }
      for (i = 0; i < t.results.length; i++) {
        convert = convert || $needsExternalization(t.results[i]);
      }
      if (!convert) {
        return v;
      }
      v.$externalizeWrapper = function() {
        var args = [], i;
        for (i = 0; i < t.params.length; i++) {
          if (t.variadic && i === t.params.length - 1) {
            var vt = t.params[i].elem, varargs = [], j;
            for (j = i; j < arguments.length; j++) {
              varargs.push($internalize(arguments[j], vt));
            }
            args.push(new (t.params[i])(varargs));
            break;
          }
          args.push($internalize(arguments[i], t.params[i]));
        }
        var result = v.apply(this, args);
        switch (t.results.length) {
        case 0:
          return;
        case 1:
          return $externalize(result, t.results[0]);
        default:
          for (i = 0; i < t.results.length; i++) {
            result[i] = $externalize(result[i], t.results[i]);
          }
          return result;
        }
      };
    }
    return v.$externalizeWrapper;
  case $kindInterface:
    if (v === $ifaceNil) {
      return null;
    }
    if (t === $packages["github.com/gopherjs/gopherjs/js"].Object || v.constructor.kind === undefined) {
      return v;
    }
    return $externalize(v.$val, v.constructor);
  case $kindMap:
    var m = {};
    var keys = $keys(v), i;
    for (i = 0; i < keys.length; i++) {
      var entry = v[keys[i]];
      m[$externalize(entry.k, t.key)] = $externalize(entry.v, t.elem);
    }
    return m;
  case $kindPtr:
    var o = {}, i;
    for (i = 0; i < t.methods.length; i++) {
      var m = t.methods[i];
      if (m[2] !== "") { /* not exported */
        continue;
      }
      (function(m) {
        o[m[1]] = $externalize(function() {
          return v[m[0]].apply(v, arguments);
        }, m[3]);
      })(m);
    }
    return o;
  case $kindSlice:
    if ($needsExternalization(t.elem)) {
      return $mapArray($sliceToArray(v), function(e) { return $externalize(e, t.elem); });
    }
    return $sliceToArray(v);
  case $kindString:
    var s = "", r, i, j = 0;
    for (i = 0; i < v.length; i += r[1], j++) {
      r = $decodeRune(v, i);
      s += String.fromCharCode(r[0]);
    }
    return s;
  case $kindStruct:
    var timePkg = $packages["time"];
    if (timePkg && v.constructor === timePkg.Time.Ptr) {
      var milli = $div64(v.UnixNano(), new $Int64(0, 1000000));
      return new Date($flatten64(milli));
    }
    var o = {}, i;
    for (i = 0; i < t.fields.length; i++) {
      var f = t.fields[i];
      if (f[2] !== "") { /* not exported */
        continue;
      }
      o[f[1]] = $externalize(v[f[0]], f[3]);
    }
    return o;
  }
  $panic(new $String("cannot externalize " + t.string));
};

var $internalize = function(v, t, recv) {
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
      var args = [], i;
      for (i = 0; i < t.params.length; i++) {
        if (t.variadic && i === t.params.length - 1) {
          var vt = t.params[i].elem, varargs = arguments[i], j;
          for (j = 0; j < varargs.$length; j++) {
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
        for (i = 0; i < t.results.length; i++) {
          result[i] = $internalize(result[i], t.results[i]);
        }
        return result;
      }
    };
  case $kindInterface:
    if (t === $packages["github.com/gopherjs/gopherjs/js"].Object) {
      return v;
    }
    if (v === null) {
      return $ifaceNil;
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
      var timePkg = $packages["time"];
      if (timePkg) {
        return new timePkg.Time(timePkg.Unix(new $Int64(0, 0), new $Int64(0, v.getTime() * 1000000)));
      }
    case Function:
      var funcType = $funcType([$sliceType($emptyInterface)], [$packages["github.com/gopherjs/gopherjs/js"].Object], true);
      return new funcType($internalize(v, funcType));
    case Number:
      return new $Float64(parseFloat(v));
    case String:
      return new $String($internalize(v, $String));
    default:
      if ($global.Node && v instanceof $global.Node) {
        return v;
      }
      var mapType = $mapType($String, $emptyInterface);
      return new mapType($internalize(v, mapType));
    }
  case $kindMap:
    var m = new $Map();
    var keys = $keys(v), i;
    for (i = 0; i < keys.length; i++) {
      var key = $internalize(keys[i], t.key);
      m[key.$key ? key.$key() : key] = { k: key, v: $internalize(v[keys[i]], t.elem) };
    }
    return m;
  case $kindSlice:
    return new t($mapArray(v, function(e) { return $internalize(e, t.elem); }));
  case $kindString:
    v = String(v);
    var s = "", i;
    for (i = 0; i < v.length; i++) {
      s += $encodeRune(v.charCodeAt(i));
    }
    return s;
  default:
    $panic(new $String("cannot internalize " + t.string));
  }
};
`
