package prelude

const typesMinified = `var $kindBool=1,$kindInt=2,$kindInt8=3,$kindInt16=4,$kindInt32=5,$kindInt64=6,$kindUint=7,$kindUint8=8,$kindUint16=9,$kindUint32=10,$kindUint64=11,$kindUintptr=12,$kindFloat32=13,$kindFloat64=14,$kindComplex64=15,$kindComplex128=16,$kindArray=17,$kindChan=18,$kindFunc=19,$kindInterface=20,$kindMap=21,$kindPtr=22,$kindSlice=23,$kindString=24,$kindStruct=25,$kindUnsafePointer=26,$methodSynthesizers=[],$addMethodSynthesizer=function(n){null!==$methodSynthesizers?$methodSynthesizers.push(n):n()},$synthesizeMethods=function(){$methodSynthesizers.forEach(function(n){n()}),$methodSynthesizers=null},$ifaceKeyFor=function(n){if(n===$ifaceNil)return"nil";var t=n.constructor;return t.string+"$"+t.keyFor(n.$val)},$identity=function(n){return n},$typeIDCounter=0,$idKey=function(n){return void 0===n.$id&&($idCounter++,n.$id=$idCounter),String(n.$id)},$newType=function(n,t,e,r,i,a,o){var $;switch(t){case $kindBool:case $kindInt:case $kindInt8:case $kindInt16:case $kindInt32:case $kindUint:case $kindUint8:case $kindUint16:case $kindUint32:case $kindUintptr:case $kindUnsafePointer:($=function(n){this.$val=n}).wrapped=!0,$.keyFor=$identity;break;case $kindString:($=function(n){this.$val=n}).wrapped=!0,$.keyFor=function(n){return"$"+n};break;case $kindFloat32:case $kindFloat64:($=function(n){this.$val=n}).wrapped=!0,$.keyFor=function(n){return $floatKey(n)};break;case $kindInt64:($=function(n,t){this.$high=n+Math.floor(Math.ceil(t)/4294967296)>>0,this.$low=t>>>0,this.$val=this}).keyFor=function(n){return n.$high+"$"+n.$low};break;case $kindUint64:($=function(n,t){this.$high=n+Math.floor(Math.ceil(t)/4294967296)>>>0,this.$low=t>>>0,this.$val=this}).keyFor=function(n){return n.$high+"$"+n.$low};break;case $kindComplex64:($=function(n,t){this.$real=$fround(n),this.$imag=$fround(t),this.$val=this}).keyFor=function(n){return n.$real+"$"+n.$imag};break;case $kindComplex128:($=function(n,t){this.$real=n,this.$imag=t,this.$val=this}).keyFor=function(n){return n.$real+"$"+n.$imag};break;case $kindArray:($=function(n){this.$val=n}).wrapped=!0,$.ptr=$newType(4,$kindPtr,"*"+e,!1,"",!1,function(n){this.$get=function(){return n},this.$set=function(n){$.copy(this,n)},this.$val=n}),$.init=function(n,t){$.elem=n,$.len=t,$.comparable=n.comparable,$.keyFor=function(t){return Array.prototype.join.call($mapArray(t,function(t){return String(n.keyFor(t)).replace(/\\/g,"\\\\").replace(/\$/g,"\\$")}),"$")},$.copy=function(t,e){$copyArray(t,e,0,0,e.length,n)},$.ptr.init($),Object.defineProperty($.ptr.nil,"nilCheck",{get:$throwNilPointerError})};break;case $kindChan:($=function(n){this.$val=n}).wrapped=!0,$.keyFor=$idKey,$.init=function(n,t,e){$.elem=n,$.sendOnly=t,$.recvOnly=e};break;case $kindFunc:($=function(n){this.$val=n}).wrapped=!0,$.init=function(n,t,e){$.params=n,$.results=t,$.variadic=e,$.comparable=!1};break;case $kindInterface:($={implementedBy:{},missingMethodFor:{}}).keyFor=$ifaceKeyFor,$.init=function(n){$.methods=n,n.forEach(function(n){$ifaceNil[n.prop]=$throwNilPointerError})};break;case $kindMap:($=function(n){this.$val=n}).wrapped=!0,$.init=function(n,t){$.key=n,$.elem=t,$.comparable=!1};break;case $kindPtr:($=o||function(n,t,e){this.$get=n,this.$set=t,this.$target=e,this.$val=this}).keyFor=$idKey,$.init=function(n){$.elem=n,$.wrapped=n.kind===$kindArray,$.nil=new $($throwNilPointerError,$throwNilPointerError)};break;case $kindSlice:($=function(n){n.constructor!==$.nativeArray&&(n=new $.nativeArray(n)),this.$array=n,this.$offset=0,this.$length=n.length,this.$capacity=n.length,this.$val=this}).init=function(n){$.elem=n,$.comparable=!1,$.nativeArray=$nativeArray(n.kind),$.nil=new $([])};break;case $kindStruct:($=function(n){this.$val=n}).wrapped=!0,$.ptr=$newType(4,$kindPtr,"*"+e,!1,i,a,o),$.ptr.elem=$,$.ptr.prototype.$get=function(){return this},$.ptr.prototype.$set=function(n){$.copy(this,n)},$.init=function(n,t){$.pkgPath=n,$.fields=t,t.forEach(function(n){n.typ.comparable||($.comparable=!1)}),$.keyFor=function(n){var e=n.$val;return $mapArray(t,function(n){return String(n.typ.keyFor(e[n.prop])).replace(/\\/g,"\\\\").replace(/\$/g,"\\$")}).join("$")},$.copy=function(n,e){for(var r=0;r<t.length;r++){var i=t[r];switch(i.typ.kind){case $kindArray:case $kindStruct:i.typ.copy(n[i.prop],e[i.prop]);continue;default:n[i.prop]=e[i.prop];continue}}};var e={};t.forEach(function(n){e[n.prop]={get:$throwNilPointerError,set:$throwNilPointerError}}),$.ptr.nil=Object.create(o.prototype,e),$.ptr.nil.$val=$.ptr.nil,$addMethodSynthesizer(function(){var n=function(n,t,e){void 0===n.prototype[t.prop]&&(n.prototype[t.prop]=function(){var n=this.$val[e.prop];return e.typ===$jsObjectPtr&&(n=new $jsObjectPtr(n)),void 0===n.$val&&(n=new e.typ(n)),n[t.prop].apply(n,arguments)})};t.forEach(function(t){t.anonymous&&($methodSet(t.typ).forEach(function(e){n($,e,t),n($.ptr,e,t)}),$methodSet($ptrType(t.typ)).forEach(function(e){n($.ptr,e,t)}))})})};break;default:$panic(new $String("invalid kind: "+t))}switch(t){case $kindBool:case $kindMap:$.zero=function(){return!1};break;case $kindInt:case $kindInt8:case $kindInt16:case $kindInt32:case $kindUint:case $kindUint8:case $kindUint16:case $kindUint32:case $kindUintptr:case $kindUnsafePointer:case $kindFloat32:case $kindFloat64:$.zero=function(){return 0};break;case $kindString:$.zero=function(){return""};break;case $kindInt64:case $kindUint64:case $kindComplex64:case $kindComplex128:var c=new $(0,0);$.zero=function(){return c};break;case $kindPtr:case $kindSlice:$.zero=function(){return $.nil};break;case $kindChan:$.zero=function(){return $chanNil};break;case $kindFunc:$.zero=function(){return $throwNilPointerError};break;case $kindInterface:$.zero=function(){return $ifaceNil};break;case $kindArray:$.zero=function(){var n=$nativeArray($.elem.kind);if(n!==Array)return new n($.len);for(var t=new Array($.len),e=0;e<$.len;e++)t[e]=$.elem.zero();return t};break;case $kindStruct:$.zero=function(){return new $.ptr};break;default:$panic(new $String("invalid kind: "+t))}return $.id=$typeIDCounter,$typeIDCounter++,$.size=n,$.kind=t,$.string=e,$.named=r,$.pkg=i,$.exported=a,$.methods=[],$.methodSetCache=null,$.comparable=!0,$},$methodSet=function(n){if(null!==n.methodSetCache)return n.methodSetCache;var t={},e=n.kind===$kindPtr;if(e&&n.elem.kind===$kindInterface)return n.methodSetCache=[],[];for(var r=[{typ:e?n.elem:n,indirect:e}],i={};r.length>0;){var a=[],o=[];r.forEach(function(n){if(!i[n.typ.string])switch(i[n.typ.string]=!0,n.typ.named&&(o=o.concat(n.typ.methods),n.indirect&&(o=o.concat($ptrType(n.typ).methods))),n.typ.kind){case $kindStruct:n.typ.fields.forEach(function(t){if(t.anonymous){var e=t.typ,r=e.kind===$kindPtr;a.push({typ:r?e.elem:e,indirect:n.indirect||r})}});break;case $kindInterface:o=o.concat(n.typ.methods)}}),o.forEach(function(n){void 0===t[n.name]&&(t[n.name]=n)}),r=a}n.methodSetCache=[];var $=Object.keys(t).sort();return 2===$.length&&"nonexported"===$[0]&&"Î¦Exported"===$[1]&&($[0]="Î¦Exported",$[1]="nonexported"),$.forEach(function(e){n.methodSetCache.push(t[e])}),n.methodSetCache},$Bool=$newType(1,$kindBool,"bool",!0,"",!1,null),$Int=$newType(4,$kindInt,"int",!0,"",!1,null),$Int8=$newType(1,$kindInt8,"int8",!0,"",!1,null),$Int16=$newType(2,$kindInt16,"int16",!0,"",!1,null),$Int32=$newType(4,$kindInt32,"int32",!0,"",!1,null),$Int64=$newType(8,$kindInt64,"int64",!0,"",!1,null),$Uint=$newType(4,$kindUint,"uint",!0,"",!1,null),$Uint8=$newType(1,$kindUint8,"uint8",!0,"",!1,null),$Uint16=$newType(2,$kindUint16,"uint16",!0,"",!1,null),$Uint32=$newType(4,$kindUint32,"uint32",!0,"",!1,null),$Uint64=$newType(8,$kindUint64,"uint64",!0,"",!1,null),$Uintptr=$newType(4,$kindUintptr,"uintptr",!0,"",!1,null),$Float32=$newType(4,$kindFloat32,"float32",!0,"",!1,null),$Float64=$newType(8,$kindFloat64,"float64",!0,"",!1,null),$Complex64=$newType(8,$kindComplex64,"complex64",!0,"",!1,null),$Complex128=$newType(16,$kindComplex128,"complex128",!0,"",!1,null),$String=$newType(8,$kindString,"string",!0,"",!1,null),$UnsafePointer=$newType(4,$kindUnsafePointer,"unsafe.Pointer",!0,"",!1,null),$nativeArray=function(n){switch(n){case $kindInt:return Int32Array;case $kindInt8:return Int8Array;case $kindInt16:return Int16Array;case $kindInt32:return Int32Array;case $kindUint:return Uint32Array;case $kindUint8:return Uint8Array;case $kindUint16:return Uint16Array;case $kindUint32:case $kindUintptr:return Uint32Array;case $kindFloat32:return Float32Array;case $kindFloat64:return Float64Array;default:return Array}},$toNativeArray=function(n,t){var e=$nativeArray(n);return e===Array?t:new e(t)},$arrayTypes={},$arrayType=function(n,t){var e=n.id+"$"+t,r=$arrayTypes[e];return void 0===r&&(r=$newType(12,$kindArray,"["+t+"]"+n.string,!1,"",!1,null),$arrayTypes[e]=r,r.init(n,t)),r},$chanType=function(n,t,e){var r=(e?"<-":"")+"chan"+(t?"<- ":" ")+n.string,i=t?"SendChan":e?"RecvChan":"Chan",a=n[i];return void 0===a&&(a=$newType(4,$kindChan,r,!1,"",!1,null),n[i]=a,a.init(n,t,e)),a},$Chan=function(n,t){(t<0||t>2147483647)&&$throwRuntimeError("makechan: size out of range"),this.$elem=n,this.$capacity=t,this.$buffer=[],this.$sendQueue=[],this.$recvQueue=[],this.$closed=!1},$chanNil=new $Chan(null,0);$chanNil.$sendQueue=$chanNil.$recvQueue={length:0,push:function(){},shift:function(){},indexOf:function(){return-1}};var $funcTypes={},$funcType=function(n,t,e){var r=$mapArray(n,function(n){return n.id}).join(",")+"$"+$mapArray(t,function(n){return n.id}).join(",")+"$"+e,i=$funcTypes[r];if(void 0===i){var a=$mapArray(n,function(n){return n.string});e&&(a[a.length-1]="..."+a[a.length-1].substr(2));var o="func("+a.join(", ")+")";1===t.length?o+=" "+t[0].string:t.length>1&&(o+=" ("+$mapArray(t,function(n){return n.string}).join(", ")+")"),i=$newType(4,$kindFunc,o,!1,"",!1,null),$funcTypes[r]=i,i.init(n,t,e)}return i},$interfaceTypes={},$interfaceType=function(n){var t=$mapArray(n,function(n){return n.pkg+","+n.name+","+n.typ.id}).join("$"),e=$interfaceTypes[t];if(void 0===e){var r="interface {}";0!==n.length&&(r="interface { "+$mapArray(n,function(n){return(""!==n.pkg?n.pkg+".":"")+n.name+n.typ.string.substr(4)}).join("; ")+" }"),e=$newType(8,$kindInterface,r,!1,"",!1,null),$interfaceTypes[t]=e,e.init(n)}return e},$emptyInterface=$interfaceType([]),$ifaceNil={},$error=$newType(8,$kindInterface,"error",!0,"",!1,null);$error.init([{prop:"Error",name:"Error",pkg:"",typ:$funcType([],[$String],!1)}]);var $mapTypes={},$mapType=function(n,t){var e=n.id+"$"+t.id,r=$mapTypes[e];return void 0===r&&(r=$newType(4,$kindMap,"map["+n.string+"]"+t.string,!1,"",!1,null),$mapTypes[e]=r,r.init(n,t)),r},$makeMap=function(n,t){for(var e={},r=0;r<t.length;r++){var i=t[r];e[n(i.k)]=i}return e},$ptrType=function(n){var t=n.ptr;return void 0===t&&(t=$newType(4,$kindPtr,"*"+n.string,!1,"",n.exported,null),n.ptr=t,t.init(n)),t},$newDataPointer=function(n,t){return t.elem.kind===$kindStruct?n:new t(function(){return n},function(t){n=t})},$indexPtr=function(n,t,e){return n.$ptr=n.$ptr||{},n.$ptr[t]||(n.$ptr[t]=new e(function(){return n[t]},function(e){n[t]=e}))},$sliceType=function(n){var t=n.slice;return void 0===t&&(t=$newType(12,$kindSlice,"[]"+n.string,!1,"",!1,null),n.slice=t,t.init(n)),t},$makeSlice=function(n,t,e){e=e||t,(t<0||t>2147483647)&&$throwRuntimeError("makeslice: len out of range"),(e<0||e<t||e>2147483647)&&$throwRuntimeError("makeslice: cap out of range");var r=new n.nativeArray(e);if(n.nativeArray===Array)for(var i=0;i<e;i++)r[i]=n.elem.zero();var a=new n(r);return a.$length=t,a},$structTypes={},$structType=function(n,t){var e=$mapArray(t,function(n){return n.name+","+n.typ.id+","+n.tag}).join("$"),r=$structTypes[e];if(void 0===r){var i="struct { "+$mapArray(t,function(n){return n.name+" "+n.typ.string+(""!==n.tag?' "'+n.tag.replace(/\\/g,"\\\\").replace(/"/g,'\\"')+'"':"")}).join("; ")+" }";0===t.length&&(i="struct {}"),r=$newType(0,$kindStruct,i,!1,"",!1,function(){this.$val=this;for(var n=0;n<t.length;n++){var e=t[n],r=arguments[n];this[e.prop]=void 0!==r?r:e.typ.zero()}}),$structTypes[e]=r,r.init(n,t)}return r},$assertType=function(n,t,e){var r,i=t.kind===$kindInterface,a="";if(n===$ifaceNil)r=!1;else if(i){var o=n.constructor.string;if(void 0===(r=t.implementedBy[o])){r=!0;for(var $=$methodSet(n.constructor),c=t.methods,u=0;u<c.length;u++){for(var p=c[u],d=!1,s=0;s<$.length;s++){var l=$[s];if(l.name===p.name&&l.pkg===p.pkg&&l.typ===p.typ){d=!0;break}}if(!d){r=!1,t.missingMethodFor[o]=p.name;break}}t.implementedBy[o]=r}r||(a=t.missingMethodFor[o])}else r=n.constructor===t;if(!r){if(e)return[t.zero(),!1];$panic(new $packages.runtime.TypeAssertionError.ptr("",n===$ifaceNil?"":n.constructor.string,t.string,a))}return i||(n=n.$val),t===$jsObjectPtr&&(n=n.object),e?[n,!0]:n};`
const types = `
var $kindBool = 1;
var $kindInt = 2;
var $kindInt8 = 3;
var $kindInt16 = 4;
var $kindInt32 = 5;
var $kindInt64 = 6;
var $kindUint = 7;
var $kindUint8 = 8;
var $kindUint16 = 9;
var $kindUint32 = 10;
var $kindUint64 = 11;
var $kindUintptr = 12;
var $kindFloat32 = 13;
var $kindFloat64 = 14;
var $kindComplex64 = 15;
var $kindComplex128 = 16;
var $kindArray = 17;
var $kindChan = 18;
var $kindFunc = 19;
var $kindInterface = 20;
var $kindMap = 21;
var $kindPtr = 22;
var $kindSlice = 23;
var $kindString = 24;
var $kindStruct = 25;
var $kindUnsafePointer = 26;

var $methodSynthesizers = [];
var $addMethodSynthesizer = function(f) {
  if ($methodSynthesizers === null) {
    f();
    return;
  }
  $methodSynthesizers.push(f);
};
var $synthesizeMethods = function() {
  $methodSynthesizers.forEach(function(f) { f(); });
  $methodSynthesizers = null;
};

var $ifaceKeyFor = function(x) {
  if (x === $ifaceNil) {
    return 'nil';
  }
  var c = x.constructor;
  return c.string + '$' + c.keyFor(x.$val);
};

var $identity = function(x) { return x; };

var $typeIDCounter = 0;

var $idKey = function(x) {
  if (x.$id === undefined) {
    $idCounter++;
    x.$id = $idCounter;
  }
  return String(x.$id);
};

var $newType = function(size, kind, string, named, pkg, exported, constructor) {
  var typ;
  switch(kind) {
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
  case $kindUnsafePointer:
    typ = function(v) { this.$val = v; };
    typ.wrapped = true;
    typ.keyFor = $identity;
    break;

  case $kindString:
    typ = function(v) { this.$val = v; };
    typ.wrapped = true;
    typ.keyFor = function(x) { return "$" + x; };
    break;

  case $kindFloat32:
  case $kindFloat64:
    typ = function(v) { this.$val = v; };
    typ.wrapped = true;
    typ.keyFor = function(x) { return $floatKey(x); };
    break;

  case $kindInt64:
    typ = function(high, low) {
      this.$high = (high + Math.floor(Math.ceil(low) / 4294967296)) >> 0;
      this.$low = low >>> 0;
      this.$val = this;
    };
    typ.keyFor = function(x) { return x.$high + "$" + x.$low; };
    break;

  case $kindUint64:
    typ = function(high, low) {
      this.$high = (high + Math.floor(Math.ceil(low) / 4294967296)) >>> 0;
      this.$low = low >>> 0;
      this.$val = this;
    };
    typ.keyFor = function(x) { return x.$high + "$" + x.$low; };
    break;

  case $kindComplex64:
    typ = function(real, imag) {
      this.$real = $fround(real);
      this.$imag = $fround(imag);
      this.$val = this;
    };
    typ.keyFor = function(x) { return x.$real + "$" + x.$imag; };
    break;

  case $kindComplex128:
    typ = function(real, imag) {
      this.$real = real;
      this.$imag = imag;
      this.$val = this;
    };
    typ.keyFor = function(x) { return x.$real + "$" + x.$imag; };
    break;

  case $kindArray:
    typ = function(v) { this.$val = v; };
    typ.wrapped = true;
    typ.ptr = $newType(4, $kindPtr, "*" + string, false, "", false, function(array) {
      this.$get = function() { return array; };
      this.$set = function(v) { typ.copy(this, v); };
      this.$val = array;
    });
    typ.init = function(elem, len) {
      typ.elem = elem;
      typ.len = len;
      typ.comparable = elem.comparable;
      typ.keyFor = function(x) {
        return Array.prototype.join.call($mapArray(x, function(e) {
          return String(elem.keyFor(e)).replace(/\\/g, "\\\\").replace(/\$/g, "\\$");
        }), "$");
      };
      typ.copy = function(dst, src) {
        $copyArray(dst, src, 0, 0, src.length, elem);
      };
      typ.ptr.init(typ);
      Object.defineProperty(typ.ptr.nil, "nilCheck", { get: $throwNilPointerError });
    };
    break;

  case $kindChan:
    typ = function(v) { this.$val = v; };
    typ.wrapped = true;
    typ.keyFor = $idKey;
    typ.init = function(elem, sendOnly, recvOnly) {
      typ.elem = elem;
      typ.sendOnly = sendOnly;
      typ.recvOnly = recvOnly;
    };
    break;

  case $kindFunc:
    typ = function(v) { this.$val = v; };
    typ.wrapped = true;
    typ.init = function(params, results, variadic) {
      typ.params = params;
      typ.results = results;
      typ.variadic = variadic;
      typ.comparable = false;
    };
    break;

  case $kindInterface:
    typ = { implementedBy: {}, missingMethodFor: {} };
    typ.keyFor = $ifaceKeyFor;
    typ.init = function(methods) {
      typ.methods = methods;
      methods.forEach(function(m) {
        $ifaceNil[m.prop] = $throwNilPointerError;
      });
    };
    break;

  case $kindMap:
    typ = function(v) { this.$val = v; };
    typ.wrapped = true;
    typ.init = function(key, elem) {
      typ.key = key;
      typ.elem = elem;
      typ.comparable = false;
    };
    break;

  case $kindPtr:
    typ = constructor || function(getter, setter, target) {
      this.$get = getter;
      this.$set = setter;
      this.$target = target;
      this.$val = this;
    };
    typ.keyFor = $idKey;
    typ.init = function(elem) {
      typ.elem = elem;
      typ.wrapped = (elem.kind === $kindArray);
      typ.nil = new typ($throwNilPointerError, $throwNilPointerError);
    };
    break;

  case $kindSlice:
    typ = function(array) {
      if (array.constructor !== typ.nativeArray) {
        array = new typ.nativeArray(array);
      }
      this.$array = array;
      this.$offset = 0;
      this.$length = array.length;
      this.$capacity = array.length;
      this.$val = this;
    };
    typ.init = function(elem) {
      typ.elem = elem;
      typ.comparable = false;
      typ.nativeArray = $nativeArray(elem.kind);
      typ.nil = new typ([]);
    };
    break;

  case $kindStruct:
    typ = function(v) { this.$val = v; };
    typ.wrapped = true;
    typ.ptr = $newType(4, $kindPtr, "*" + string, false, pkg, exported, constructor);
    typ.ptr.elem = typ;
    typ.ptr.prototype.$get = function() { return this; };
    typ.ptr.prototype.$set = function(v) { typ.copy(this, v); };
    typ.init = function(pkgPath, fields) {
      typ.pkgPath = pkgPath;
      typ.fields = fields;
      fields.forEach(function(f) {
        if (!f.typ.comparable) {
          typ.comparable = false;
        }
      });
      typ.keyFor = function(x) {
        var val = x.$val;
        return $mapArray(fields, function(f) {
          return String(f.typ.keyFor(val[f.prop])).replace(/\\/g, "\\\\").replace(/\$/g, "\\$");
        }).join("$");
      };
      typ.copy = function(dst, src) {
        for (var i = 0; i < fields.length; i++) {
          var f = fields[i];
          switch (f.typ.kind) {
          case $kindArray:
          case $kindStruct:
            f.typ.copy(dst[f.prop], src[f.prop]);
            continue;
          default:
            dst[f.prop] = src[f.prop];
            continue;
          }
        }
      };
      /* nil value */
      var properties = {};
      fields.forEach(function(f) {
        properties[f.prop] = { get: $throwNilPointerError, set: $throwNilPointerError };
      });
      typ.ptr.nil = Object.create(constructor.prototype, properties);
      typ.ptr.nil.$val = typ.ptr.nil;
      /* methods for embedded fields */
      $addMethodSynthesizer(function() {
        var synthesizeMethod = function(target, m, f) {
          if (target.prototype[m.prop] !== undefined) { return; }
          target.prototype[m.prop] = function() {
            var v = this.$val[f.prop];
            if (f.typ === $jsObjectPtr) {
              v = new $jsObjectPtr(v);
            }
            if (v.$val === undefined) {
              v = new f.typ(v);
            }
            return v[m.prop].apply(v, arguments);
          };
        };
        fields.forEach(function(f) {
          if (f.anonymous) {
            $methodSet(f.typ).forEach(function(m) {
              synthesizeMethod(typ, m, f);
              synthesizeMethod(typ.ptr, m, f);
            });
            $methodSet($ptrType(f.typ)).forEach(function(m) {
              synthesizeMethod(typ.ptr, m, f);
            });
          }
        });
      });
    };
    break;

  default:
    $panic(new $String("invalid kind: " + kind));
  }

  switch (kind) {
  case $kindBool:
  case $kindMap:
    typ.zero = function() { return false; };
    break;

  case $kindInt:
  case $kindInt8:
  case $kindInt16:
  case $kindInt32:
  case $kindUint:
  case $kindUint8 :
  case $kindUint16:
  case $kindUint32:
  case $kindUintptr:
  case $kindUnsafePointer:
  case $kindFloat32:
  case $kindFloat64:
    typ.zero = function() { return 0; };
    break;

  case $kindString:
    typ.zero = function() { return ""; };
    break;

  case $kindInt64:
  case $kindUint64:
  case $kindComplex64:
  case $kindComplex128:
    var zero = new typ(0, 0);
    typ.zero = function() { return zero; };
    break;

  case $kindPtr:
  case $kindSlice:
    typ.zero = function() { return typ.nil; };
    break;

  case $kindChan:
    typ.zero = function() { return $chanNil; };
    break;

  case $kindFunc:
    typ.zero = function() { return $throwNilPointerError; };
    break;

  case $kindInterface:
    typ.zero = function() { return $ifaceNil; };
    break;

  case $kindArray:
    typ.zero = function() {
      var arrayClass = $nativeArray(typ.elem.kind);
      if (arrayClass !== Array) {
        return new arrayClass(typ.len);
      }
      var array = new Array(typ.len);
      for (var i = 0; i < typ.len; i++) {
        array[i] = typ.elem.zero();
      }
      return array;
    };
    break;

  case $kindStruct:
    typ.zero = function() { return new typ.ptr(); };
    break;

  default:
    $panic(new $String("invalid kind: " + kind));
  }

  typ.id = $typeIDCounter;
  $typeIDCounter++;
  typ.size = size;
  typ.kind = kind;
  typ.string = string;
  typ.named = named;
  typ.pkg = pkg;
  typ.exported = exported;
  typ.methods = [];
  typ.methodSetCache = null;
  typ.comparable = true;
  return typ;
};

var $methodSet = function(typ) {
  if (typ.methodSetCache !== null) {
    return typ.methodSetCache;
  }
  var base = {};

  var isPtr = (typ.kind === $kindPtr);
  if (isPtr && typ.elem.kind === $kindInterface) {
    typ.methodSetCache = [];
    return [];
  }

  var current = [{typ: isPtr ? typ.elem : typ, indirect: isPtr}];

  var seen = {};

  while (current.length > 0) {
    var next = [];
    var mset = [];

    current.forEach(function(e) {
      if (seen[e.typ.string]) {
        return;
      }
      seen[e.typ.string] = true;

      if (e.typ.named) {
        mset = mset.concat(e.typ.methods);
        if (e.indirect) {
          mset = mset.concat($ptrType(e.typ).methods);
        }
      }

      switch (e.typ.kind) {
      case $kindStruct:
        e.typ.fields.forEach(function(f) {
          if (f.anonymous) {
            var fTyp = f.typ;
            var fIsPtr = (fTyp.kind === $kindPtr);
            next.push({typ: fIsPtr ? fTyp.elem : fTyp, indirect: e.indirect || fIsPtr});
          }
        });
        break;

      case $kindInterface:
        mset = mset.concat(e.typ.methods);
        break;
      }
    });

    mset.forEach(function(m) {
      if (base[m.name] === undefined) {
        base[m.name] = m;
      }
    });

    current = next;
  }

  typ.methodSetCache = [];
  Object.keys(base).sort().forEach(function(name) {
    typ.methodSetCache.push(base[name]);
  });
  return typ.methodSetCache;
};

var $Bool          = $newType( 1, $kindBool,          "bool",           true, "", false, null);
var $Int           = $newType( 4, $kindInt,           "int",            true, "", false, null);
var $Int8          = $newType( 1, $kindInt8,          "int8",           true, "", false, null);
var $Int16         = $newType( 2, $kindInt16,         "int16",          true, "", false, null);
var $Int32         = $newType( 4, $kindInt32,         "int32",          true, "", false, null);
var $Int64         = $newType( 8, $kindInt64,         "int64",          true, "", false, null);
var $Uint          = $newType( 4, $kindUint,          "uint",           true, "", false, null);
var $Uint8         = $newType( 1, $kindUint8,         "uint8",          true, "", false, null);
var $Uint16        = $newType( 2, $kindUint16,        "uint16",         true, "", false, null);
var $Uint32        = $newType( 4, $kindUint32,        "uint32",         true, "", false, null);
var $Uint64        = $newType( 8, $kindUint64,        "uint64",         true, "", false, null);
var $Uintptr       = $newType( 4, $kindUintptr,       "uintptr",        true, "", false, null);
var $Float32       = $newType( 4, $kindFloat32,       "float32",        true, "", false, null);
var $Float64       = $newType( 8, $kindFloat64,       "float64",        true, "", false, null);
var $Complex64     = $newType( 8, $kindComplex64,     "complex64",      true, "", false, null);
var $Complex128    = $newType(16, $kindComplex128,    "complex128",     true, "", false, null);
var $String        = $newType( 8, $kindString,        "string",         true, "", false, null);
var $UnsafePointer = $newType( 4, $kindUnsafePointer, "unsafe.Pointer", true, "", false, null);

var $nativeArray = function(elemKind) {
  switch (elemKind) {
  case $kindInt:
    return Int32Array;
  case $kindInt8:
    return Int8Array;
  case $kindInt16:
    return Int16Array;
  case $kindInt32:
    return Int32Array;
  case $kindUint:
    return Uint32Array;
  case $kindUint8:
    return Uint8Array;
  case $kindUint16:
    return Uint16Array;
  case $kindUint32:
    return Uint32Array;
  case $kindUintptr:
    return Uint32Array;
  case $kindFloat32:
    return Float32Array;
  case $kindFloat64:
    return Float64Array;
  default:
    return Array;
  }
};
var $toNativeArray = function(elemKind, array) {
  var nativeArray = $nativeArray(elemKind);
  if (nativeArray === Array) {
    return array;
  }
  return new nativeArray(array);
};
var $arrayTypes = {};
var $arrayType = function(elem, len) {
  var typeKey = elem.id + "$" + len;
  var typ = $arrayTypes[typeKey];
  if (typ === undefined) {
    typ = $newType(12, $kindArray, "[" + len + "]" + elem.string, false, "", false, null);
    $arrayTypes[typeKey] = typ;
    typ.init(elem, len);
  }
  return typ;
};

var $chanType = function(elem, sendOnly, recvOnly) {
  var string = (recvOnly ? "<-" : "") + "chan" + (sendOnly ? "<- " : " ") + elem.string;
  var field = sendOnly ? "SendChan" : (recvOnly ? "RecvChan" : "Chan");
  var typ = elem[field];
  if (typ === undefined) {
    typ = $newType(4, $kindChan, string, false, "", false, null);
    elem[field] = typ;
    typ.init(elem, sendOnly, recvOnly);
  }
  return typ;
};
var $Chan = function(elem, capacity) {
  if (capacity < 0 || capacity > 2147483647) {
    $throwRuntimeError("makechan: size out of range");
  }
  this.$elem = elem;
  this.$capacity = capacity;
  this.$buffer = [];
  this.$sendQueue = [];
  this.$recvQueue = [];
  this.$closed = false;
};
var $chanNil = new $Chan(null, 0);
$chanNil.$sendQueue = $chanNil.$recvQueue = { length: 0, push: function() {}, shift: function() { return undefined; }, indexOf: function() { return -1; } };

var $funcTypes = {};
var $funcType = function(params, results, variadic) {
  var typeKey = $mapArray(params, function(p) { return p.id; }).join(",") + "$" + $mapArray(results, function(r) { return r.id; }).join(",") + "$" + variadic;
  var typ = $funcTypes[typeKey];
  if (typ === undefined) {
    var paramTypes = $mapArray(params, function(p) { return p.string; });
    if (variadic) {
      paramTypes[paramTypes.length - 1] = "..." + paramTypes[paramTypes.length - 1].substr(2);
    }
    var string = "func(" + paramTypes.join(", ") + ")";
    if (results.length === 1) {
      string += " " + results[0].string;
    } else if (results.length > 1) {
      string += " (" + $mapArray(results, function(r) { return r.string; }).join(", ") + ")";
    }
    typ = $newType(4, $kindFunc, string, false, "", false, null);
    $funcTypes[typeKey] = typ;
    typ.init(params, results, variadic);
  }
  return typ;
};

var $interfaceTypes = {};
var $interfaceType = function(methods) {
  var typeKey = $mapArray(methods, function(m) { return m.pkg + "," + m.name + "," + m.typ.id; }).join("$");
  var typ = $interfaceTypes[typeKey];
  if (typ === undefined) {
    var string = "interface {}";
    if (methods.length !== 0) {
      string = "interface { " + $mapArray(methods, function(m) {
        return (m.pkg !== "" ? m.pkg + "." : "") + m.name + m.typ.string.substr(4);
      }).join("; ") + " }";
    }
    typ = $newType(8, $kindInterface, string, false, "", false, null);
    $interfaceTypes[typeKey] = typ;
    typ.init(methods);
  }
  return typ;
};
var $emptyInterface = $interfaceType([]);
var $ifaceNil = {};
var $error = $newType(8, $kindInterface, "error", true, "", false, null);
$error.init([{prop: "Error", name: "Error", pkg: "", typ: $funcType([], [$String], false)}]);

var $mapTypes = {};
var $mapType = function(key, elem) {
  var typeKey = key.id + "$" + elem.id;
  var typ = $mapTypes[typeKey];
  if (typ === undefined) {
    typ = $newType(4, $kindMap, "map[" + key.string + "]" + elem.string, false, "", false, null);
    $mapTypes[typeKey] = typ;
    typ.init(key, elem);
  }
  return typ;
};
var $makeMap = function(keyForFunc, entries) {
  var m = {};
  for (var i = 0; i < entries.length; i++) {
    var e = entries[i];
    m[keyForFunc(e.k)] = e;
  }
  return m;
};

var $ptrType = function(elem) {
  var typ = elem.ptr;
  if (typ === undefined) {
    typ = $newType(4, $kindPtr, "*" + elem.string, false, "", elem.exported, null);
    elem.ptr = typ;
    typ.init(elem);
  }
  return typ;
};

var $newDataPointer = function(data, constructor) {
  if (constructor.elem.kind === $kindStruct) {
    return data;
  }
  return new constructor(function() { return data; }, function(v) { data = v; });
};

var $indexPtr = function(array, index, constructor) {
  array.$ptr = array.$ptr || {};
  return array.$ptr[index] || (array.$ptr[index] = new constructor(function() { return array[index]; }, function(v) { array[index] = v; }));
};

var $sliceType = function(elem) {
  var typ = elem.slice;
  if (typ === undefined) {
    typ = $newType(12, $kindSlice, "[]" + elem.string, false, "", false, null);
    elem.slice = typ;
    typ.init(elem);
  }
  return typ;
};
var $makeSlice = function(typ, length, capacity) {
  capacity = capacity || length;
  if (length < 0 || length > 2147483647) {
    $throwRuntimeError("makeslice: len out of range");
  }
  if (capacity < 0 || capacity < length || capacity > 2147483647) {
    $throwRuntimeError("makeslice: cap out of range");
  }
  var array = new typ.nativeArray(capacity);
  if (typ.nativeArray === Array) {
    for (var i = 0; i < capacity; i++) {
      array[i] = typ.elem.zero();
    }
  }
  var slice = new typ(array);
  slice.$length = length;
  return slice;
};

var $structTypes = {};
var $structType = function(pkgPath, fields) {
  var typeKey = $mapArray(fields, function(f) { return f.name + "," + f.typ.id + "," + f.tag; }).join("$");
  var typ = $structTypes[typeKey];
  if (typ === undefined) {
    var string = "struct { " + $mapArray(fields, function(f) {
      return f.name + " " + f.typ.string + (f.tag !== "" ? (" \"" + f.tag.replace(/\\/g, "\\\\").replace(/"/g, "\\\"") + "\"") : "");
    }).join("; ") + " }";
    if (fields.length === 0) {
      string = "struct {}";
    }
    typ = $newType(0, $kindStruct, string, false, "", false, function() {
      this.$val = this;
      for (var i = 0; i < fields.length; i++) {
        var f = fields[i];
        var arg = arguments[i];
        this[f.prop] = arg !== undefined ? arg : f.typ.zero();
      }
    });
    $structTypes[typeKey] = typ;
    typ.init(pkgPath, fields);
  }
  return typ;
};

var $assertType = function(value, type, returnTuple) {
  var isInterface = (type.kind === $kindInterface), ok, missingMethod = "";
  if (value === $ifaceNil) {
    ok = false;
  } else if (!isInterface) {
    ok = value.constructor === type;
  } else {
    var valueTypeString = value.constructor.string;
    ok = type.implementedBy[valueTypeString];
    if (ok === undefined) {
      ok = true;
      var valueMethodSet = $methodSet(value.constructor);
      var interfaceMethods = type.methods;
      for (var i = 0; i < interfaceMethods.length; i++) {
        var tm = interfaceMethods[i];
        var found = false;
        for (var j = 0; j < valueMethodSet.length; j++) {
          var vm = valueMethodSet[j];
          if (vm.name === tm.name && vm.pkg === tm.pkg && vm.typ === tm.typ) {
            found = true;
            break;
          }
        }
        if (!found) {
          ok = false;
          type.missingMethodFor[valueTypeString] = tm.name;
          break;
        }
      }
      type.implementedBy[valueTypeString] = ok;
    }
    if (!ok) {
      missingMethod = type.missingMethodFor[valueTypeString];
    }
  }

  if (!ok) {
    if (returnTuple) {
      return [type.zero(), false];
    }
    $panic(new $packages["runtime"].TypeAssertionError.ptr("", (value === $ifaceNil ? "" : value.constructor.string), type.string, missingMethod));
  }

  if (!isInterface) {
    value = value.$val;
  }
  if (type === $jsObjectPtr) {
    value = value.object;
  }
  return returnTuple ? [value, true] : value;
};
`
