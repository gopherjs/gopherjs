package prelude

const Prelude = prelude + numeric + types + goroutines + jsmapping

// Minified is Prelude ran through UglifyJS 3 via https://skalman.github.io/UglifyJS-online/
// with default options.
const Minified = `var $global,$module;if(Error.stackTraceLimit=1/0,"undefined"!=typeof window?$global=window:"undefined"!=typeof self?$global=self:"undefined"!=typeof global?($global=global).require=require:$global=this,void 0===$global||void 0===$global.Array)throw new Error("no global object found");"undefined"!=typeof module&&($module=module);var $throwRuntimeError,$packages={},$idCounter=0,$keys=function(e){return e?Object.keys(e):[]},$flushConsole=function(){},$throwNilPointerError=function(){$throwRuntimeError("invalid memory address or nil pointer dereference")},$call=function(e,n,r){return e.apply(n,r)},$makeFunc=function(e){return function(){return $externalize(e(this,new($sliceType($jsObjectPtr))($global.Array.prototype.slice.call(arguments,[]))),$emptyInterface)}},$unused=function(e){},$mapArray=function(e,n){for(var r=new e.constructor(e.length),t=0;t<e.length;t++)r[t]=n(e[t]);return r},$methodVal=function(e,n){var r=e.$methodVals||{};e.$methodVals=r;var t=r[n];if(void 0!==t)return t;var i=e[n];return t=function(){$stackDepthOffset--;try{return i.apply(e,arguments)}finally{$stackDepthOffset++}},r[n]=t,t},$methodExpr=function(e,n){var r=e.prototype[n];return void 0===r.$expr&&(r.$expr=function(){$stackDepthOffset--;try{return e.wrapped&&(arguments[0]=new e(arguments[0])),Function.call.apply(r,arguments)}finally{$stackDepthOffset++}}),r.$expr},$ifaceMethodExprs={},$ifaceMethodExpr=function(e){var n=$ifaceMethodExprs["$"+e];return void 0===n&&(n=$ifaceMethodExprs["$"+e]=function(){$stackDepthOffset--;try{return Function.call.apply(arguments[0][e],arguments)}finally{$stackDepthOffset++}}),n},$subslice=function(e,n,r,t){(n<0||r<n||t<r||r>e.$capacity||t>e.$capacity)&&$throwRuntimeError("slice bounds out of range");var i=new e.constructor(e.$array);return i.$offset=e.$offset+n,i.$length=e.$length-n,i.$capacity=e.$capacity-n,void 0!==r&&(i.$length=r-n),void 0!==t&&(i.$capacity=t-n),i},$substring=function(e,n,r){return(n<0||r<n||r>e.length)&&$throwRuntimeError("slice bounds out of range"),e.substring(n,r)},$sliceToArray=function(e){return e.$array.constructor!==Array?e.$array.subarray(e.$offset,e.$offset+e.$length):e.$array.slice(e.$offset,e.$offset+e.$length)},$decodeRune=function(e,n){var r=e.charCodeAt(n);if(r<128)return[r,1];if(r!=r||r<192)return[65533,1];var t=e.charCodeAt(n+1);if(t!=t||t<128||192<=t)return[65533,1];if(r<224)return(a=(31&r)<<6|63&t)<=127?[65533,1]:[a,2];var i=e.charCodeAt(n+2);if(i!=i||i<128||192<=i)return[65533,1];if(r<240)return(a=(15&r)<<12|(63&t)<<6|63&i)<=2047?[65533,1]:55296<=a&&a<=57343?[65533,1]:[a,3];var a,o=e.charCodeAt(n+3);return o!=o||o<128||192<=o?[65533,1]:r<248?(a=(7&r)<<18|(63&t)<<12|(63&i)<<6|63&o)<=65535||1114111<a?[65533,1]:[a,4]:[65533,1]},$encodeRune=function(e){return(e<0||e>1114111||55296<=e&&e<=57343)&&(e=65533),e<=127?String.fromCharCode(e):e<=2047?String.fromCharCode(192|e>>6,128|63&e):e<=65535?String.fromCharCode(224|e>>12,128|e>>6&63,128|63&e):String.fromCharCode(240|e>>18,128|e>>12&63,128|e>>6&63,128|63&e)},$stringToBytes=function(e){for(var n=new Uint8Array(e.length),r=0;r<e.length;r++)n[r]=e.charCodeAt(r);return n},$bytesToString=function(e){if(0===e.$length)return"";for(var n="",r=0;r<e.$length;r+=1e4)n+=String.fromCharCode.apply(void 0,e.$array.subarray(e.$offset+r,e.$offset+Math.min(e.$length,r+1e4)));return n},$stringToRunes=function(e){for(var n,r=new Int32Array(e.length),t=0,i=0;i<e.length;i+=n[1],t++)n=$decodeRune(e,i),r[t]=n[0];return r.subarray(0,t)},$runesToString=function(e){if(0===e.$length)return"";for(var n="",r=0;r<e.$length;r++)n+=$encodeRune(e.$array[e.$offset+r]);return n},$copyString=function(e,n){for(var r=Math.min(n.length,e.$length),t=0;t<r;t++)e.$array[e.$offset+t]=n.charCodeAt(t);return r},$copySlice=function(e,n){var r=Math.min(n.$length,e.$length);return $copyArray(e.$array,n.$array,e.$offset,n.$offset,r,e.constructor.elem),r},$copyArray=function(e,n,r,t,i,a){if(0!==i&&(e!==n||r!==t))if(n.subarray)e.set(n.subarray(t,t+i),r);else{switch(a.kind){case $kindArray:case $kindStruct:if(e===n&&r>t){for(var o=i-1;o>=0;o--)a.copy(e[r+o],n[t+o]);return}for(o=0;o<i;o++)a.copy(e[r+o],n[t+o]);return}if(e===n&&r>t)for(o=i-1;o>=0;o--)e[r+o]=n[t+o];else for(o=0;o<i;o++)e[r+o]=n[t+o]}},$clone=function(e,n){var r=n.zero();return n.copy(r,e),r},$pointerOfStructConversion=function(e,n){void 0===e.$proxies&&(e.$proxies={},e.$proxies[e.constructor.string]=e);var r=e.$proxies[n.string];if(void 0===r){for(var t={},i=0;i<n.elem.fields.length;i++)!function(n){t[n]={get:function(){return e[n]},set:function(r){e[n]=r}}}(n.elem.fields[i].prop);(r=Object.create(n.prototype,t)).$val=r,e.$proxies[n.string]=r,r.$proxies=e.$proxies}return r},$append=function(e){return $internalAppend(e,arguments,1,arguments.length-1)},$appendSlice=function(e,n){if(n.constructor===String){var r=$stringToBytes(n);return $internalAppend(e,r,0,r.length)}return $internalAppend(e,n.$array,n.$offset,n.$length)},$internalAppend=function(e,n,r,t){if(0===t)return e;var i=e.$array,a=e.$offset,o=e.$length+t,$=e.$capacity;if(o>$)if(a=0,$=Math.max(o,e.$capacity<1024?2*e.$capacity:Math.floor(5*e.$capacity/4)),e.$array.constructor===Array){(i=e.$array.slice(e.$offset,e.$offset+e.$length)).length=$;for(var c=e.constructor.elem.zero,u=e.$length;u<$;u++)i[u]=c()}else(i=new e.$array.constructor($)).set(e.$array.subarray(e.$offset,e.$offset+e.$length));$copyArray(i,n,a+e.$length,r,t,e.constructor.elem);var l=new e.constructor(i);return l.$offset=a,l.$length=o,l.$capacity=$,l},$equal=function(e,n,r){if(r===$jsObjectPtr)return e===n;switch(r.kind){case $kindComplex64:case $kindComplex128:return e.$real===n.$real&&e.$imag===n.$imag;case $kindInt64:case $kindUint64:return e.$high===n.$high&&e.$low===n.$low;case $kindArray:if(e.length!==n.length)return!1;for(var t=0;t<e.length;t++)if(!$equal(e[t],n[t],r.elem))return!1;return!0;case $kindStruct:for(t=0;t<r.fields.length;t++){var i=r.fields[t];if(!$equal(e[i.prop],n[i.prop],i.typ))return!1}return!0;case $kindInterface:return $interfaceIsEqual(e,n);default:return e===n}},$interfaceIsEqual=function(e,n){return e===$ifaceNil||n===$ifaceNil?e===n:e.constructor===n.constructor&&(e.constructor===$jsObjectPtr?e.object===n.object:(e.constructor.comparable||$throwRuntimeError("comparing uncomparable type "+e.constructor.string),$equal(e.$val,n.$val,e.constructor)))},$min=Math.min,$mod=function(e,n){return e%n},$parseInt=parseInt,$parseFloat=function(e){return null!=e&&e.constructor===Number?e:parseFloat(e)},$froundBuf=new Float32Array(1),$fround=Math.fround||function(e){return $froundBuf[0]=e,$froundBuf[0]},$imul=Math.imul||function(e,n){var r=65535&e,t=65535&n;return r*t+((e>>>16&65535)*t+r*(n>>>16&65535)<<16>>>0)>>0},$floatKey=function(e){return e!=e?"NaN$"+ ++$idCounter:String(e)},$flatten64=function(e){return 4294967296*e.$high+e.$low},$shiftLeft64=function(e,n){return 0===n?e:n<32?new e.constructor(e.$high<<n|e.$low>>>32-n,e.$low<<n>>>0):n<64?new e.constructor(e.$low<<n-32,0):new e.constructor(0,0)},$shiftRightInt64=function(e,n){return 0===n?e:n<32?new e.constructor(e.$high>>n,(e.$low>>>n|e.$high<<32-n)>>>0):n<64?new e.constructor(e.$high>>31,e.$high>>n-32>>>0):e.$high<0?new e.constructor(-1,4294967295):new e.constructor(0,0)},$shiftRightUint64=function(e,n){return 0===n?e:n<32?new e.constructor(e.$high>>>n,(e.$low>>>n|e.$high<<32-n)>>>0):n<64?new e.constructor(0,e.$high>>>n-32):new e.constructor(0,0)},$mul64=function(e,n){var r=0,t=0;0!=(1&n.$low)&&(r=e.$high,t=e.$low);for(var i=1;i<32;i++)0!=(n.$low&1<<i)&&(r+=e.$high<<i|e.$low>>>32-i,t+=e.$low<<i>>>0);for(i=0;i<32;i++)0!=(n.$high&1<<i)&&(r+=e.$low<<i);return new e.constructor(r,t)},$div64=function(e,n,r){0===n.$high&&0===n.$low&&$throwRuntimeError("integer divide by zero");var t=1,i=1,a=e.$high,o=e.$low;a<0&&(t=-1,i=-1,a=-a,0!==o&&(a--,o=4294967296-o));var $=n.$high,c=n.$low;n.$high<0&&(t*=-1,$=-$,0!==c&&($--,c=4294967296-c));for(var u=0,l=0,s=0;$<2147483648&&(a>$||a===$&&o>c);)$=($<<1|c>>>31)>>>0,c=c<<1>>>0,s++;for(var f=0;f<=s;f++)u=u<<1|l>>>31,l=l<<1>>>0,(a>$||a===$&&o>=c)&&(a-=$,(o-=c)<0&&(a--,o+=4294967296),4294967296===++l&&(u++,l=0)),c=(c>>>1|$<<31)>>>0,$>>>=1;return r?new e.constructor(a*i,o*i):new e.constructor(u*t,l*t)},$divComplex=function(e,n){var r=e.$real===1/0||e.$real===-1/0||e.$imag===1/0||e.$imag===-1/0,t=n.$real===1/0||n.$real===-1/0||n.$imag===1/0||n.$imag===-1/0,i=!r&&(e.$real!=e.$real||e.$imag!=e.$imag),a=!t&&(n.$real!=n.$real||n.$imag!=n.$imag);if(i||a)return new e.constructor(NaN,NaN);if(r&&!t)return new e.constructor(1/0,1/0);if(!r&&t)return new e.constructor(0,0);if(0===n.$real&&0===n.$imag)return 0===e.$real&&0===e.$imag?new e.constructor(NaN,NaN):new e.constructor(1/0,1/0);if(Math.abs(n.$real)<=Math.abs(n.$imag)){var o=n.$real/n.$imag,$=n.$real*o+n.$imag;return new e.constructor((e.$real*o+e.$imag)/$,(e.$imag*o-e.$real)/$)}o=n.$imag/n.$real,$=n.$imag*o+n.$real;return new e.constructor((e.$imag*o+e.$real)/$,(e.$imag-e.$real*o)/$)},$kindBool=1,$kindInt=2,$kindInt8=3,$kindInt16=4,$kindInt32=5,$kindInt64=6,$kindUint=7,$kindUint8=8,$kindUint16=9,$kindUint32=10,$kindUint64=11,$kindUintptr=12,$kindFloat32=13,$kindFloat64=14,$kindComplex64=15,$kindComplex128=16,$kindArray=17,$kindChan=18,$kindFunc=19,$kindInterface=20,$kindMap=21,$kindPtr=22,$kindSlice=23,$kindString=24,$kindStruct=25,$kindUnsafePointer=26,$methodSynthesizers=[],$addMethodSynthesizer=function(e){null!==$methodSynthesizers?$methodSynthesizers.push(e):e()},$synthesizeMethods=function(){$methodSynthesizers.forEach(function(e){e()}),$methodSynthesizers=null},$ifaceKeyFor=function(e){if(e===$ifaceNil)return"nil";var n=e.constructor;return n.string+"$"+n.keyFor(e.$val)},$identity=function(e){return e},$typeIDCounter=0,$idKey=function(e){return void 0===e.$id&&($idCounter++,e.$id=$idCounter),String(e.$id)},$newType=function(e,n,r,t,i,a,o){var $;switch(n){case $kindBool:case $kindInt:case $kindInt8:case $kindInt16:case $kindInt32:case $kindUint:case $kindUint8:case $kindUint16:case $kindUint32:case $kindUintptr:case $kindUnsafePointer:($=function(e){this.$val=e}).wrapped=!0,$.keyFor=$identity;break;case $kindString:($=function(e){this.$val=e}).wrapped=!0,$.keyFor=function(e){return"$"+e};break;case $kindFloat32:case $kindFloat64:($=function(e){this.$val=e}).wrapped=!0,$.keyFor=function(e){return $floatKey(e)};break;case $kindInt64:($=function(e,n){this.$high=e+Math.floor(Math.ceil(n)/4294967296)>>0,this.$low=n>>>0,this.$val=this}).keyFor=function(e){return e.$high+"$"+e.$low};break;case $kindUint64:($=function(e,n){this.$high=e+Math.floor(Math.ceil(n)/4294967296)>>>0,this.$low=n>>>0,this.$val=this}).keyFor=function(e){return e.$high+"$"+e.$low};break;case $kindComplex64:($=function(e,n){this.$real=$fround(e),this.$imag=$fround(n),this.$val=this}).keyFor=function(e){return e.$real+"$"+e.$imag};break;case $kindComplex128:($=function(e,n){this.$real=e,this.$imag=n,this.$val=this}).keyFor=function(e){return e.$real+"$"+e.$imag};break;case $kindArray:($=function(e){this.$val=e}).wrapped=!0,$.ptr=$newType(4,$kindPtr,"*"+r,!1,"",!1,function(e){this.$get=function(){return e},this.$set=function(e){$.copy(this,e)},this.$val=e}),$.init=function(e,n){$.elem=e,$.len=n,$.comparable=e.comparable,$.keyFor=function(n){return Array.prototype.join.call($mapArray(n,function(n){return String(e.keyFor(n)).replace(/\\/g,"\\\\").replace(/\$/g,"\\$")}),"$")},$.copy=function(n,r){$copyArray(n,r,0,0,r.length,e)},$.ptr.init($),Object.defineProperty($.ptr.nil,"nilCheck",{get:$throwNilPointerError})};break;case $kindChan:($=function(e){this.$val=e}).wrapped=!0,$.keyFor=$idKey,$.init=function(e,n,r){$.elem=e,$.sendOnly=n,$.recvOnly=r};break;case $kindFunc:($=function(e){this.$val=e}).wrapped=!0,$.init=function(e,n,r){$.params=e,$.results=n,$.variadic=r,$.comparable=!1};break;case $kindInterface:($={implementedBy:{},missingMethodFor:{}}).keyFor=$ifaceKeyFor,$.init=function(e){$.methods=e,e.forEach(function(e){$ifaceNil[e.prop]=$throwNilPointerError})};break;case $kindMap:($=function(e){this.$val=e}).wrapped=!0,$.init=function(e,n){$.key=e,$.elem=n,$.comparable=!1};break;case $kindPtr:($=o||function(e,n,r){this.$get=e,this.$set=n,this.$target=r,this.$val=this}).keyFor=$idKey,$.init=function(e){$.elem=e,$.wrapped=e.kind===$kindArray,$.nil=new $($throwNilPointerError,$throwNilPointerError)};break;case $kindSlice:($=function(e){e.constructor!==$.nativeArray&&(e=new $.nativeArray(e)),this.$array=e,this.$offset=0,this.$length=e.length,this.$capacity=e.length,this.$val=this}).init=function(e){$.elem=e,$.comparable=!1,$.nativeArray=$nativeArray(e.kind),$.nil=new $([])};break;case $kindStruct:($=function(e){this.$val=e}).wrapped=!0,$.ptr=$newType(4,$kindPtr,"*"+r,!1,i,a,o),$.ptr.elem=$,$.ptr.prototype.$get=function(){return this},$.ptr.prototype.$set=function(e){$.copy(this,e)},$.init=function(e,n){$.pkgPath=e,$.fields=n,n.forEach(function(e){e.typ.comparable||($.comparable=!1)}),$.keyFor=function(e){var r=e.$val;return $mapArray(n,function(e){return String(e.typ.keyFor(r[e.prop])).replace(/\\/g,"\\\\").replace(/\$/g,"\\$")}).join("$")},$.copy=function(e,r){for(var t=0;t<n.length;t++){var i=n[t];switch(i.typ.kind){case $kindArray:case $kindStruct:i.typ.copy(e[i.prop],r[i.prop]);continue;default:e[i.prop]=r[i.prop];continue}}};var r={};n.forEach(function(e){r[e.prop]={get:$throwNilPointerError,set:$throwNilPointerError}}),$.ptr.nil=Object.create(o.prototype,r),$.ptr.nil.$val=$.ptr.nil,$addMethodSynthesizer(function(){var e=function(e,n,r){void 0===e.prototype[n.prop]&&(e.prototype[n.prop]=function(){var e=this.$val[r.prop];return r.typ===$jsObjectPtr&&(e=new $jsObjectPtr(e)),void 0===e.$val&&(e=new r.typ(e)),e[n.prop].apply(e,arguments)})};n.forEach(function(n){n.anonymous&&($methodSet(n.typ).forEach(function(r){e($,r,n),e($.ptr,r,n)}),$methodSet($ptrType(n.typ)).forEach(function(r){e($.ptr,r,n)}))})})};break;default:$panic(new $String("invalid kind: "+n))}switch(n){case $kindBool:case $kindMap:$.zero=function(){return!1};break;case $kindInt:case $kindInt8:case $kindInt16:case $kindInt32:case $kindUint:case $kindUint8:case $kindUint16:case $kindUint32:case $kindUintptr:case $kindUnsafePointer:case $kindFloat32:case $kindFloat64:$.zero=function(){return 0};break;case $kindString:$.zero=function(){return""};break;case $kindInt64:case $kindUint64:case $kindComplex64:case $kindComplex128:var c=new $(0,0);$.zero=function(){return c};break;case $kindPtr:case $kindSlice:$.zero=function(){return $.nil};break;case $kindChan:$.zero=function(){return $chanNil};break;case $kindFunc:$.zero=function(){return $throwNilPointerError};break;case $kindInterface:$.zero=function(){return $ifaceNil};break;case $kindArray:$.zero=function(){var e=$nativeArray($.elem.kind);if(e!==Array)return new e($.len);for(var n=new Array($.len),r=0;r<$.len;r++)n[r]=$.elem.zero();return n};break;case $kindStruct:$.zero=function(){return new $.ptr};break;default:$panic(new $String("invalid kind: "+n))}return $.id=$typeIDCounter,$typeIDCounter++,$.size=e,$.kind=n,$.string=r,$.named=t,$.pkg=i,$.exported=a,$.methods=[],$.methodSetCache=null,$.comparable=!0,$},$methodSet=function(e){if(null!==e.methodSetCache)return e.methodSetCache;var n={},r=e.kind===$kindPtr;if(r&&e.elem.kind===$kindInterface)return e.methodSetCache=[],[];for(var t=[{typ:r?e.elem:e,indirect:r}],i={};t.length>0;){var a=[],o=[];t.forEach(function(e){if(!i[e.typ.string])switch(i[e.typ.string]=!0,e.typ.named&&(o=o.concat(e.typ.methods),e.indirect&&(o=o.concat($ptrType(e.typ).methods))),e.typ.kind){case $kindStruct:e.typ.fields.forEach(function(n){if(n.anonymous){var r=n.typ,t=r.kind===$kindPtr;a.push({typ:t?r.elem:r,indirect:e.indirect||t})}});break;case $kindInterface:o=o.concat(e.typ.methods)}}),o.forEach(function(e){void 0===n[e.name]&&(n[e.name]=e)}),t=a}return e.methodSetCache=[],Object.keys(n).sort().forEach(function(r){e.methodSetCache.push(n[r])}),e.methodSetCache},$Bool=$newType(1,$kindBool,"bool",!0,"",!1,null),$Int=$newType(4,$kindInt,"int",!0,"",!1,null),$Int8=$newType(1,$kindInt8,"int8",!0,"",!1,null),$Int16=$newType(2,$kindInt16,"int16",!0,"",!1,null),$Int32=$newType(4,$kindInt32,"int32",!0,"",!1,null),$Int64=$newType(8,$kindInt64,"int64",!0,"",!1,null),$Uint=$newType(4,$kindUint,"uint",!0,"",!1,null),$Uint8=$newType(1,$kindUint8,"uint8",!0,"",!1,null),$Uint16=$newType(2,$kindUint16,"uint16",!0,"",!1,null),$Uint32=$newType(4,$kindUint32,"uint32",!0,"",!1,null),$Uint64=$newType(8,$kindUint64,"uint64",!0,"",!1,null),$Uintptr=$newType(4,$kindUintptr,"uintptr",!0,"",!1,null),$Float32=$newType(4,$kindFloat32,"float32",!0,"",!1,null),$Float64=$newType(8,$kindFloat64,"float64",!0,"",!1,null),$Complex64=$newType(8,$kindComplex64,"complex64",!0,"",!1,null),$Complex128=$newType(16,$kindComplex128,"complex128",!0,"",!1,null),$String=$newType(8,$kindString,"string",!0,"",!1,null),$UnsafePointer=$newType(4,$kindUnsafePointer,"unsafe.Pointer",!0,"",!1,null),$nativeArray=function(e){switch(e){case $kindInt:return Int32Array;case $kindInt8:return Int8Array;case $kindInt16:return Int16Array;case $kindInt32:return Int32Array;case $kindUint:return Uint32Array;case $kindUint8:return Uint8Array;case $kindUint16:return Uint16Array;case $kindUint32:case $kindUintptr:return Uint32Array;case $kindFloat32:return Float32Array;case $kindFloat64:return Float64Array;default:return Array}},$toNativeArray=function(e,n){var r=$nativeArray(e);return r===Array?n:new r(n)},$arrayTypes={},$arrayType=function(e,n){var r=e.id+"$"+n,t=$arrayTypes[r];return void 0===t&&(t=$newType(12,$kindArray,"["+n+"]"+e.string,!1,"",!1,null),$arrayTypes[r]=t,t.init(e,n)),t},$chanType=function(e,n,r){var t=(r?"<-":"")+"chan"+(n?"<- ":" ")+e.string,i=n?"SendChan":r?"RecvChan":"Chan",a=e[i];return void 0===a&&(a=$newType(4,$kindChan,t,!1,"",!1,null),e[i]=a,a.init(e,n,r)),a},$Chan=function(e,n){(n<0||n>2147483647)&&$throwRuntimeError("makechan: size out of range"),this.$elem=e,this.$capacity=n,this.$buffer=[],this.$sendQueue=[],this.$recvQueue=[],this.$closed=!1},$chanNil=new $Chan(null,0);$chanNil.$sendQueue=$chanNil.$recvQueue={length:0,push:function(){},shift:function(){},indexOf:function(){return-1}};var $funcTypes={},$funcType=function(e,n,r){var t=$mapArray(e,function(e){return e.id}).join(",")+"$"+$mapArray(n,function(e){return e.id}).join(",")+"$"+r,i=$funcTypes[t];if(void 0===i){var a=$mapArray(e,function(e){return e.string});r&&(a[a.length-1]="..."+a[a.length-1].substr(2));var o="func("+a.join(", ")+")";1===n.length?o+=" "+n[0].string:n.length>1&&(o+=" ("+$mapArray(n,function(e){return e.string}).join(", ")+")"),i=$newType(4,$kindFunc,o,!1,"",!1,null),$funcTypes[t]=i,i.init(e,n,r)}return i},$interfaceTypes={},$interfaceType=function(e){var n=$mapArray(e,function(e){return e.pkg+","+e.name+","+e.typ.id}).join("$"),r=$interfaceTypes[n];if(void 0===r){var t="interface {}";0!==e.length&&(t="interface { "+$mapArray(e,function(e){return(""!==e.pkg?e.pkg+".":"")+e.name+e.typ.string.substr(4)}).join("; ")+" }"),r=$newType(8,$kindInterface,t,!1,"",!1,null),$interfaceTypes[n]=r,r.init(e)}return r},$emptyInterface=$interfaceType([]),$ifaceNil={},$error=$newType(8,$kindInterface,"error",!0,"",!1,null);$error.init([{prop:"Error",name:"Error",pkg:"",typ:$funcType([],[$String],!1)}]);var $panicValue,$jsObjectPtr,$jsErrorPtr,$mapTypes={},$mapType=function(e,n){var r=e.id+"$"+n.id,t=$mapTypes[r];return void 0===t&&(t=$newType(4,$kindMap,"map["+e.string+"]"+n.string,!1,"",!1,null),$mapTypes[r]=t,t.init(e,n)),t},$makeMap=function(e,n){for(var r={},t=0;t<n.length;t++){var i=n[t];r[e(i.k)]=i}return r},$ptrType=function(e){var n=e.ptr;return void 0===n&&(n=$newType(4,$kindPtr,"*"+e.string,!1,"",e.exported,null),e.ptr=n,n.init(e)),n},$newDataPointer=function(e,n){return n.elem.kind===$kindStruct?e:new n(function(){return e},function(n){e=n})},$indexPtr=function(e,n,r){return e.$ptr=e.$ptr||{},e.$ptr[n]||(e.$ptr[n]=new r(function(){return e[n]},function(r){e[n]=r}))},$sliceType=function(e){var n=e.slice;return void 0===n&&(n=$newType(12,$kindSlice,"[]"+e.string,!1,"",!1,null),e.slice=n,n.init(e)),n},$makeSlice=function(e,n,r){r=r||n,(n<0||n>2147483647)&&$throwRuntimeError("makeslice: len out of range"),(r<0||r<n||r>2147483647)&&$throwRuntimeError("makeslice: cap out of range");var t=new e.nativeArray(r);if(e.nativeArray===Array)for(var i=0;i<r;i++)t[i]=e.elem.zero();var a=new e(t);return a.$length=n,a},$structTypes={},$structType=function(e,n){var r=$mapArray(n,function(e){return e.name+","+e.typ.id+","+e.tag}).join("$"),t=$structTypes[r];if(void 0===t){var i="struct { "+$mapArray(n,function(e){return e.name+" "+e.typ.string+(""!==e.tag?' "'+e.tag.replace(/\\/g,"\\\\").replace(/"/g,'\\"')+'"':"")}).join("; ")+" }";0===n.length&&(i="struct {}"),t=$newType(0,$kindStruct,i,!1,"",!1,function(){this.$val=this;for(var e=0;e<n.length;e++){var r=n[e],t=arguments[e];this[r.prop]=void 0!==t?t:r.typ.zero()}}),$structTypes[r]=t,t.init(e,n)}return t},$assertType=function(e,n,r){var t,i=n.kind===$kindInterface,a="";if(e===$ifaceNil)t=!1;else if(i){var o=e.constructor.string;if(void 0===(t=n.implementedBy[o])){t=!0;for(var $=$methodSet(e.constructor),c=n.methods,u=0;u<c.length;u++){for(var l=c[u],s=!1,f=0;f<$.length;f++){var d=$[f];if(d.name===l.name&&d.pkg===l.pkg&&d.typ===l.typ){s=!0;break}}if(!s){t=!1,n.missingMethodFor[o]=l.name;break}}n.implementedBy[o]=t}t||(a=n.missingMethodFor[o])}else t=e.constructor===n;if(!t){if(r)return[n.zero(),!1];$panic(new $packages.runtime.TypeAssertionError.ptr("",e===$ifaceNil?"":e.constructor.string,n.string,a))}return i||(e=e.$val),n===$jsObjectPtr&&(e=e.object),r?[e,!0]:e},$stackDepthOffset=0,$getStackDepth=function(){var e=new Error;if(void 0!==e.stack)return $stackDepthOffset+e.stack.split("\n").length},$panicStackDepth=null,$callDeferred=function(e,n,r){if(!r&&null!==e&&e.index>=$curGoroutine.deferStack.length)throw n;if(null!==n){var t=null;try{$curGoroutine.deferStack.push(e),$panic(new $jsErrorPtr(n))}catch(e){t=e}return $curGoroutine.deferStack.pop(),void $callDeferred(e,t)}if(!$curGoroutine.asleep){$stackDepthOffset--;var i=$panicStackDepth,a=$panicValue,o=$curGoroutine.panicStack.pop();void 0!==o&&($panicStackDepth=$getStackDepth(),$panicValue=o);try{for(;;){if(null===e&&void 0===(e=$curGoroutine.deferStack[$curGoroutine.deferStack.length-1])){if($panicStackDepth=null,o.Object instanceof Error)throw o.Object;var $;throw $=o.constructor===$String?o.$val:void 0!==o.Error?o.Error():void 0!==o.String?o.String():o,new Error($)}var c=e.pop();if(void 0===c){if($curGoroutine.deferStack.pop(),void 0!==o){e=null;continue}return}var u=c[0].apply(c[2],c[1]);if(u&&void 0!==u.$blk){if(e.push([u.$blk,[],u]),r)throw null;return}if(void 0!==o&&null===$panicStackDepth)throw null}}finally{void 0!==o&&(null!==$panicStackDepth&&$curGoroutine.panicStack.push(o),$panicStackDepth=i,$panicValue=a),$stackDepthOffset++}}},$panic=function(e){$curGoroutine.panicStack.push(e),$callDeferred(null,null,!0)},$recover=function(){return null===$panicStackDepth||void 0!==$panicStackDepth&&$panicStackDepth!==$getStackDepth()-2?$ifaceNil:($panicStackDepth=null,$panicValue)},$throw=function(e){throw e},$noGoroutine={asleep:!1,exit:!1,deferStack:[],panicStack:[]},$curGoroutine=$noGoroutine,$totalGoroutines=0,$awakeGoroutines=0,$checkForDeadlock=!0,$mainFinished=!1,$go=function(e,n){$totalGoroutines++,$awakeGoroutines++;var r=function(){try{$curGoroutine=r;var t=e.apply(void 0,n);if(t&&void 0!==t.$blk)return e=function(){return t.$blk()},void(n=[]);r.exit=!0}catch(e){if(!r.exit)throw e}finally{$curGoroutine=$noGoroutine,r.exit&&($totalGoroutines--,r.asleep=!0),r.asleep&&($awakeGoroutines--,!$mainFinished&&0===$awakeGoroutines&&$checkForDeadlock&&(console.error("fatal error: all goroutines are asleep - deadlock!"),void 0!==$global.process&&$global.process.exit(2)))}};r.asleep=!1,r.exit=!1,r.deferStack=[],r.panicStack=[],$schedule(r)},$scheduled=[],$runScheduled=function(){try{for(var e;void 0!==(e=$scheduled.shift());)e()}finally{$scheduled.length>0&&setTimeout($runScheduled,0)}},$schedule=function(e){e.asleep&&(e.asleep=!1,$awakeGoroutines++),$scheduled.push(e),$curGoroutine===$noGoroutine&&$runScheduled()},$setTimeout=function(e,n){return $awakeGoroutines++,setTimeout(function(){$awakeGoroutines--,e()},n)},$block=function(){$curGoroutine===$noGoroutine&&$throwRuntimeError("cannot block in JavaScript callback, fix by wrapping code in goroutine"),$curGoroutine.asleep=!0},$send=function(e,n){e.$closed&&$throwRuntimeError("send on closed channel");var r=e.$recvQueue.shift();if(void 0===r){if(!(e.$buffer.length<e.$capacity)){var t,i=$curGoroutine;return e.$sendQueue.push(function(e){return t=e,$schedule(i),n}),$block(),{$blk:function(){t&&$throwRuntimeError("send on closed channel")}}}e.$buffer.push(n)}else r([n,!0])},$recv=function(e){var n=e.$sendQueue.shift();void 0!==n&&e.$buffer.push(n(!1));var r=e.$buffer.shift();if(void 0!==r)return[r,!0];if(e.$closed)return[e.$elem.zero(),!1];var t=$curGoroutine,i={$blk:function(){return this.value}};return e.$recvQueue.push(function(e){i.value=e,$schedule(t)}),$block(),i},$close=function(e){for(e.$closed&&$throwRuntimeError("close of closed channel"),e.$closed=!0;;){var n=e.$sendQueue.shift();if(void 0===n)break;n(!0)}for(;;){var r=e.$recvQueue.shift();if(void 0===r)break;r([e.$elem.zero(),!1])}},$select=function(e){for(var n=[],r=-1,t=0;t<e.length;t++){var i,a=(i=e[t])[0];switch(i.length){case 0:r=t;break;case 1:(0!==a.$sendQueue.length||0!==a.$buffer.length||a.$closed)&&n.push(t);break;case 2:a.$closed&&$throwRuntimeError("send on closed channel"),(0!==a.$recvQueue.length||a.$buffer.length<a.$capacity)&&n.push(t)}}if(0!==n.length&&(r=n[Math.floor(Math.random()*n.length)]),-1!==r)switch((i=e[r]).length){case 0:return[r];case 1:return[r,$recv(i[0])];case 2:return $send(i[0],i[1]),[r]}var o=[],$=$curGoroutine,c={$blk:function(){return this.selection}},u=function(){for(var e=0;e<o.length;e++){var n=o[e],r=n[0],t=r.indexOf(n[1]);-1!==t&&r.splice(t,1)}};for(t=0;t<e.length;t++)!function(n){var r=e[n];switch(r.length){case 1:var t=function(e){c.selection=[n,e],u(),$schedule($)};o.push([r[0].$recvQueue,t]),r[0].$recvQueue.push(t);break;case 2:t=function(){return r[0].$closed&&$throwRuntimeError("send on closed channel"),c.selection=[n],u(),$schedule($),r[1]};o.push([r[0].$sendQueue,t]),r[0].$sendQueue.push(t)}}(t);return $block(),c},$needsExternalization=function(e){switch(e.kind){case $kindBool:case $kindInt:case $kindInt8:case $kindInt16:case $kindInt32:case $kindUint:case $kindUint8:case $kindUint16:case $kindUint32:case $kindUintptr:case $kindFloat32:case $kindFloat64:return!1;default:return e!==$jsObjectPtr}},$externalize=function(e,n){if(n===$jsObjectPtr)return e;switch(n.kind){case $kindBool:case $kindInt:case $kindInt8:case $kindInt16:case $kindInt32:case $kindUint:case $kindUint8:case $kindUint16:case $kindUint32:case $kindUintptr:case $kindFloat32:case $kindFloat64:return e;case $kindInt64:case $kindUint64:return $flatten64(e);case $kindArray:return $needsExternalization(n.elem)?$mapArray(e,function(e){return $externalize(e,n.elem)}):e;case $kindFunc:return $externalizeFunction(e,n,!1);case $kindInterface:return e===$ifaceNil?null:e.constructor===$jsObjectPtr?e.$val.object:$externalize(e.$val,e.constructor);case $kindMap:for(var r={},t=$keys(e),i=0;i<t.length;i++){var a=e[t[i]];r[$externalize(a.k,n.key)]=$externalize(a.v,n.elem)}return r;case $kindPtr:return e===n.nil?null:$externalize(e.$get(),n.elem);case $kindSlice:return $needsExternalization(n.elem)?$mapArray($sliceToArray(e),function(e){return $externalize(e,n.elem)}):$sliceToArray(e);case $kindString:if($isASCII(e))return e;var o,$="";for(i=0;i<e.length;i+=o[1]){var c=(o=$decodeRune(e,i))[0];if(c>65535){var u=Math.floor((c-65536)/1024)+55296,l=(c-65536)%1024+56320;$+=String.fromCharCode(u,l)}else $+=String.fromCharCode(c)}return $;case $kindStruct:var s=$packages.time;if(void 0!==s&&e.constructor===s.Time.ptr){var f=$div64(e.UnixNano(),new $Int64(0,1e6));return new Date($flatten64(f))}var d={},p=function(e,n){if(n===$jsObjectPtr)return e;switch(n.kind){case $kindPtr:return e===n.nil?d:p(e.$get(),n.elem);case $kindStruct:var r=n.fields[0];return p(e[r.prop],r.typ);case $kindInterface:return p(e.$val,e.constructor);default:return d}},h=p(e,n);if(h!==d)return h;h={};for(i=0;i<n.fields.length;i++){var k=n.fields[i];k.exported&&(h[k.name]=$externalize(e[k.prop],k.typ))}return h}$throwRuntimeError("cannot externalize "+n.string)},$externalizeFunction=function(e,n,r){return e===$throwNilPointerError?null:(void 0===e.$externalizeWrapper&&($checkForDeadlock=!1,e.$externalizeWrapper=function(){for(var t=[],i=0;i<n.params.length;i++){if(n.variadic&&i===n.params.length-1){for(var a=n.params[i].elem,o=[],$=i;$<arguments.length;$++)o.push($internalize(arguments[$],a));t.push(new n.params[i](o));break}t.push($internalize(arguments[i],n.params[i]))}var c=e.apply(r?this:void 0,t);switch(n.results.length){case 0:return;case 1:return $externalize(c,n.results[0]);default:for(i=0;i<n.results.length;i++)c[i]=$externalize(c[i],n.results[i]);return c}}),e.$externalizeWrapper)},$internalize=function(e,n,r){if(n===$jsObjectPtr)return e;if(n===$jsObjectPtr.elem&&$throwRuntimeError("cannot internalize js.Object, use *js.Object instead"),e&&void 0!==e.__internal_object__)return $assertType(e.__internal_object__,n,!1);var t=$packages.time;if(void 0!==t&&n===t.Time)return null!=e&&e.constructor===Date||$throwRuntimeError("cannot internalize time.Time from "+typeof e+", must be Date"),t.Unix(new $Int64(0,0),new $Int64(0,1e6*e.getTime()));switch(n.kind){case $kindBool:return!!e;case $kindInt:return parseInt(e);case $kindInt8:return parseInt(e)<<24>>24;case $kindInt16:return parseInt(e)<<16>>16;case $kindInt32:return parseInt(e)>>0;case $kindUint:return parseInt(e);case $kindUint8:return parseInt(e)<<24>>>24;case $kindUint16:return parseInt(e)<<16>>>16;case $kindUint32:case $kindUintptr:return parseInt(e)>>>0;case $kindInt64:case $kindUint64:return new n(0,e);case $kindFloat32:case $kindFloat64:return parseFloat(e);case $kindArray:return e.length!==n.len&&$throwRuntimeError("got array with wrong size from JavaScript native"),$mapArray(e,function(e){return $internalize(e,n.elem)});case $kindFunc:return function(){for(var t=[],i=0;i<n.params.length;i++){if(n.variadic&&i===n.params.length-1){for(var a=n.params[i].elem,o=arguments[i],$=0;$<o.$length;$++)t.push($externalize(o.$array[o.$offset+$],a));break}t.push($externalize(arguments[i],n.params[i]))}var c=e.apply(r,t);switch(n.results.length){case 0:return;case 1:return $internalize(c,n.results[0]);default:for(i=0;i<n.results.length;i++)c[i]=$internalize(c[i],n.results[i]);return c}};case $kindInterface:if(0!==n.methods.length&&$throwRuntimeError("cannot internalize "+n.string),null===e)return $ifaceNil;if(void 0===e)return new $jsObjectPtr(void 0);switch(e.constructor){case Int8Array:return new($sliceType($Int8))(e);case Int16Array:return new($sliceType($Int16))(e);case Int32Array:return new($sliceType($Int))(e);case Uint8Array:return new($sliceType($Uint8))(e);case Uint16Array:return new($sliceType($Uint16))(e);case Uint32Array:return new($sliceType($Uint))(e);case Float32Array:return new($sliceType($Float32))(e);case Float64Array:return new($sliceType($Float64))(e);case Array:return $internalize(e,$sliceType($emptyInterface));case Boolean:return new $Bool(!!e);case Date:return void 0===t?new $jsObjectPtr(e):new t.Time($internalize(e,t.Time));case Function:var i=$funcType([$sliceType($emptyInterface)],[$jsObjectPtr],!0);return new i($internalize(e,i));case Number:return new $Float64(parseFloat(e));case String:return new $String($internalize(e,$String));default:if($global.Node&&e instanceof $global.Node)return new $jsObjectPtr(e);var a=$mapType($String,$emptyInterface);return new a($internalize(e,a))}case $kindMap:for(var o={},$=$keys(e),c=0;c<$.length;c++){var u=$internalize($[c],n.key);o[n.key.keyFor(u)]={k:u,v:$internalize(e[$[c]],n.elem)}}return o;case $kindPtr:if(n.elem.kind===$kindStruct)return $internalize(e,n.elem);case $kindSlice:return new n($mapArray(e,function(e){return $internalize(e,n.elem)}));case $kindString:if(e=String(e),$isASCII(e))return e;var l="";for(c=0;c<e.length;){var s=e.charCodeAt(c);if(55296<=s&&s<=56319){var f=e.charCodeAt(c+1);l+=$encodeRune(1024*(s-55296)+f-56320+65536),c+=2}else l+=$encodeRune(s),c++}return l;case $kindStruct:var d={},p=function(n){if(n===$jsObjectPtr)return e;switch(n===$jsObjectPtr.elem&&$throwRuntimeError("cannot internalize js.Object, use *js.Object instead"),n.kind){case $kindPtr:return p(n.elem);case $kindStruct:var r=n.fields[0],t=p(r.typ);if(t!==d){var i=new n.ptr;return i[r.prop]=t,i}return d;default:return d}},h=p(n);if(h!==d)return h}$throwRuntimeError("cannot internalize "+n.string)},$isASCII=function(e){for(var n=0;n<e.length;n++)if(e.charCodeAt(n)>=128)return!1;return!0};`

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
  if (low < 0 || high < low || max < high || high > slice.$capacity || max > slice.$capacity) {
    $throwRuntimeError("slice bounds out of range");
  }
  var s = new slice.constructor(slice.$array);
  s.$offset = slice.$offset + low;
  s.$length = slice.$length - low;
  s.$capacity = slice.$capacity - low;
  if (high !== undefined) {
    s.$length = high - low;
  }
  if (max !== undefined) {
    s.$capacity = max - low;
  }
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
