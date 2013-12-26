package translator

func init() {
	natives["reflect"] = `
    go$reflect = {
      rtype: rtype.Ptr, uncommonType: uncommonType.Ptr, method: method.Ptr, arrayType: arrayType.Ptr, chanType: chanType.Ptr, funcType: funcType.Ptr, interfaceType: interfaceType.Ptr, mapType: mapType.Ptr, ptrType: ptrType.Ptr, sliceType: sliceType.Ptr, structType: structType.Ptr,
      imethod: imethod.Ptr, structField: structField.Ptr,
      kinds: { Bool: go$pkg.Bool, Int: go$pkg.Int, Int8: go$pkg.Int8, Int16: go$pkg.Int16, Int32: go$pkg.Int32, Int64: go$pkg.Int64, Uint: go$pkg.Uint, Uint8: go$pkg.Uint8, Uint16: go$pkg.Uint16, Uint32: go$pkg.Uint32, Uint64: go$pkg.Uint64, Uintptr: go$pkg.Uintptr, Float32: go$pkg.Float32, Float64: go$pkg.Float64, Complex64: go$pkg.Complex64, Complex128: go$pkg.Complex128, Array: go$pkg.Array, Chan: go$pkg.Chan, Func: go$pkg.Func, Interface: go$pkg.Interface, Map: go$pkg.Map, Ptr: go$pkg.Ptr, Slice: go$pkg.Slice, String: go$pkg.String, Struct: go$pkg.Struct, UnsafePointer: go$pkg.UnsafePointer },
      RecvDir: go$pkg.RecvDir, SendDir: go$pkg.SendDir, BothDir: go$pkg.BothDir
    };

    var fieldName = function(field) {
      if (field.name === Go$StringPointer.nil) {
        var ntyp = field.typ;
        if (ntyp.Kind() === go$pkg.Ptr) {
          ntyp = ntyp.Elem().common();
        }
        return ntyp.Name();
      }
      return field.name.go$get();
    };
    var copyStruct = function(dst, src, typ) {
      var fields = typ.structType.fields.array, i;
      for (i = 0; i < fields.length; i += 1) {
        var field = fields[i];
        var name = fieldName(field);
        if (field.typ.Kind() === go$pkg.Struct) {
          console.log("ARG");
        }
        dst[name] = src[name];
      }
    };

    TypeOf = function(i) {
      if (i === null) {
        return null;
      }
      return i.constructor.reflectType();
    };
    ValueOf = function(i) {
      if (i === null) {
        return new Value.Ptr();
      }
      var typ = i.constructor.reflectType();
      return new Value.Ptr(typ, i.go$val, typ.Kind() << flagKindShift);
    };
    Zero = function(typ) {
      var val;
      switch (typ.Kind()) {
      case go$pkg.Bool:
        val = false;
        break;
      case go$pkg.Int:
      case go$pkg.Int8:
      case go$pkg.Int16:
      case go$pkg.Int32:
      case go$pkg.Uint:
      case go$pkg.Uint8:
      case go$pkg.Uint16:
      case go$pkg.Uint32:
      case go$pkg.Uintptr:
      case go$pkg.Float32:
      case go$pkg.Float64:
        val = 0;
        break;
      case go$pkg.Int64:
      case go$pkg.Uint64:
      case go$pkg.Complex64:
      case go$pkg.Complex128:
        val = new typ.alg(0, 0);
        break;
      case go$pkg.Array:
        var elemType = typ.Elem();
        val = go$makeNativeArray(elemType.alg.kind, typ.Len(), function() { return Zero(elemType).val; });
        break;
      case go$pkg.Interface:
        val = null;
        break;
      case go$pkg.Map:
        val = false;
        break;
      case go$pkg.Ptr:
      case go$pkg.Slice:
        val = typ.alg.nil;
        break;
      case go$pkg.String:
        val = "";
        break;
      case go$pkg.Struct:
        val = new typ.alg.Ptr();
        break;
      default:
        throw new Go$Panic("reflect.Zero(" + typ.string.go$get() + "): type not yet supported");
      }
      return new Value.Ptr(typ, val, typ.Kind() << flagKindShift);
    };
    New = function(typ) {
      var ptrType = typ.common().ptrTo();
      if (typ.Kind() === go$pkg.Struct) {
        return new Value.Ptr(ptrType, new typ.alg.Ptr(), go$pkg.Ptr << flagKindShift);
      }
      return new Value.Ptr(ptrType, go$newDataPointer(Zero(typ).val, ptrType.alg), go$pkg.Ptr << flagKindShift);
    };
    MakeSlice = function(typ, len, cap) {
      if (typ.Kind() !== go$pkg.Slice) {
        throw new Go$Panic("reflect.MakeSlice of non-slice type");
      }
      if (len < 0) {
        throw new Go$Panic("reflect.MakeSlice: negative len");
      }
      if (cap < 0) {
        throw new Go$Panic("reflect.MakeSlice: negative cap");
      }
      if (len > cap) {
        throw new Go$Panic("reflect.MakeSlice: len > cap");
      }
      return new Value.Ptr(typ.common(), typ.alg.make(len, cap, function() { return Zero(typ.Elem()).val; }), go$pkg.Slice << flagKindShift);
    };
    makemap = function(t) {
      return new Go$Map();
    };
    mapaccess = function(t, m, key) {
      var entry = m[key];
      if (entry === undefined) {
        return [undefined, false];
      }
      return [entry.v, true];
    };
    mapassign = function(t, m, key, val, ok) {
      if (t.Elem().kind === go$pkg.Struct) {
        var newVal = {};
        copyStruct(newVal, val, t.Elem());
        val = newVal;
      }
      m[key] = { k: key, v: val }; // FIXME key
    };
    maplen = function(m) {
      return go$keys(m).length;
    };
    mapiterinit = function(t, m) {
      return [m, go$keys(m), 0];
    };
    mapiterkey = function(it) {
      var key = it[1][it[2]];
      return [it[0][key].k, true];
    };
    mapiternext = function(it) {
      it[2] += 1;
    };
    valueInterface = function(v, safe) {
      if (v.flag === 0) {
        throw new Go$Panic(new ValueError.Ptr("reflect.Value.Interface", 0));
      }
      if (safe && (v.flag & flagRO) !== 0) {
        throw new Go$Panic("reflect.Value.Interface: cannot return value obtained from unexported field or method")
      }
      var val = v.iword();
      if (v.typ.Kind() === go$pkg.Interface || val.constructor === v.typ.alg) {
        return val;
      }
      if (v.typ.Kind() === go$pkg.Ptr) {
        return new v.typ.alg(val.go$get, val.go$set);
      }
      return new v.typ.alg(val);
    };
    methodName = function() {
      return "?FIXME?";
    };
    Copy = function(dst, src) {
      var dk = dst.kind();
      if (dk !== go$pkg.Array && dk !== go$pkg.Slice) {
        throw new Go$Panic(new ValueError.Ptr("reflect.Copy", dk));
      }
      if (dk === go$pkg.Array) {
        dst.mustBeAssignable();
      }
      dst.mustBeExported();

      var sk = src.kind();
      if (sk !== go$pkg.Array && sk != go$pkg.Slice) {
        throw new Go$Panic(new ValueError.Ptr("reflect.Copy", sk));
      }
      src.mustBeExported();

      typesMustMatch("reflect.Copy", dst.typ.Elem(), src.typ.Elem());

      var n = Math.min(src.Len(), dst.Len());
      var i;
      for(i = 0; i < n; i += 1) {
        dst.Index(i).Set(src.Index(i));
      }
      return n;
    };

    rtype.Ptr.prototype.ptrTo = function() {
      return go$ptrType(this.alg).reflectType();
    };

    Value.Ptr.prototype.iword = function() {
      if ((this.flag & flagIndir) !== 0 && this.typ.Kind() !== go$pkg.Struct) {
        return this.val.go$get();
      }
      return this.val;
    };
    Value.Ptr.prototype.Bytes = function() {
      this.mustBe(go$pkg.Slice);
      if (this.typ.Elem().Kind() !== go$pkg.Uint8) {
        throw new Go$Panic("reflect.Value.Bytes of non-byte slice");
      }
      return this.val;
    };
    Value.Ptr.prototype.call = function(op, args) {
      if (this.val === null) {
        throw new Go$Panic("reflect.Value.Call: call of nil function");
      }

      var isSlice = (op === "CallSlice");
      var t = this.typ;
      var n = t.NumIn();
      if (isSlice) {
        if (!t.IsVariadic()) {
          throw new Go$Panic("reflect: CallSlice of non-variadic function");
        }
        if (args.length < n) {
          throw new Go$Panic("reflect: CallSlice with too few input arguments");
        }
        if (args.length > n) {
          throw new Go$Panic("reflect: CallSlice with too many input arguments");
        }
      } else {
        if (t.IsVariadic()) {
          n -= 1;
        }
        if (args.length < n) {
          throw new Go$Panic("reflect: Call with too few input arguments");
        }
        if (!t.IsVariadic() && args.length > n) {
          throw new Go$Panic("reflect: Call with too many input arguments");
        }
      }
      var i;
      for (i = 0; i < args.length; i += 1) {
        if (args.array[args.offset + i].Kind() === go$pkg.Invalid) {
          throw new Go$Panic("reflect: " + op + " using zero Value argument");
        }
      }
      for (i = 0; i < n; i += 1) {
        var xt = args.array[args.offset + i].Type(), targ = t.In(i);
        if (!xt.AssignableTo(targ)) {
          throw new Go$Panic("reflect: " + op + " using " + xt.String() + " as type " + targ.String());
        }
      }

      var argsArray = new Array(n);
      for (i = 0; i < n; i += 1) {
        argsArray[i] = args.array[args.offset + i].iword();
      }
      var results = this.val.apply(null, argsArray);
      if (t.NumOut() === 0) {
        results = [];
      } else if (t.NumOut() === 1) {
        results = [results];
      }
      for (i = 0; i < t.NumOut(); i += 1) {
        var typ = t.Out(i);
        var flag = typ.Kind() << flagKindShift;
        results[i] = new Value.Ptr(typ, results[i], flag);
      }
      return new (go$sliceType(Value))(results);
    };
    Value.Ptr.prototype.Cap = function() {
      var k = this.kind();
      switch (k) {
      case go$pkg.Slice:
        return this.iword().capacity;
      }
      throw new Go$Panic(new ValueError.Ptr("reflect.Value.Cap", k));
    };
    Value.Ptr.prototype.Elem = function() {
      switch (this.kind()) {
      case go$pkg.Interface:
        var val = this.iword();
        if (val === null) {
          return new Value.Ptr();
        }
        var typ = val.constructor.reflectType();
        var fl = this.flag & flagRO;
        fl |= typ.Kind() << flagKindShift;
        return new Value.Ptr(typ, val.go$val, fl);

      case go$pkg.Ptr:
        var val = this.iword();
        if (this.IsNil()) {
          return new Value.Ptr();
        }
        var tt = this.typ.ptrType;
        var fl = (this.flag & flagRO) | flagIndir | flagAddr;
        fl |= tt.elem.Kind() << flagKindShift;
        return new Value.Ptr(tt.elem, val, fl);
      }
      throw new Go$Panic(new ValueError.Ptr("reflect.Value.Elem", this.kind()));
    };
    Value.Ptr.prototype.Field = function(i) {
      this.mustBe(go$pkg.Struct);
      var tt = this.typ.structType;
      if (i < 0 || i >= tt.fields.length) {
        throw new Go$Panic("reflect: Field index out of range");
      }
      var field = tt.fields.array[i];
      var name = fieldName(field);
      var typ = field.typ;
      var fl = this.flag & (flagRO | flagIndir | flagAddr);
      // if (field.pkgPath !== nil) {
      //  fl |= flagRO
      // }
      fl |= typ.Kind() << flagKindShift;
      if ((this.flag & flagIndir) !== 0 && typ.Kind() !== go$pkg.Struct) {
        var struct = this.val;
        return new Value.Ptr(typ, new (go$ptrType(typ))(function() { return struct[name]; }, function(v) { struct[name] = v; }), fl);
      }
      return new Value.Ptr(typ, this.val[name], fl);
    };
    Value.Ptr.prototype.Index = function(i) {
      var k = this.kind();
      switch (k) {
      case go$pkg.Array:
        var tt = this.typ.arrayType;
        if (i < 0 || i >= tt.len) {
          throw new Go$Panic("reflect: array index out of range");
        }
        var typ = tt.elem;
        var fl = this.flag & (flagRO | flagIndir | flagAddr);
        fl |= typ.Kind() << flagKindShift;
        if ((this.flag & flagIndir) !== 0 && typ.Kind() !== go$pkg.Struct) {
          var array = this.val.go$get();
          return new Value.Ptr(typ, new (go$ptrType(typ))(function() { return array[i]; }, function(v) { array[i] = v; }), fl);
        }
        return new Value.Ptr(typ, this.iword()[i], fl);
      case go$pkg.Slice:
        if (i < 0 || i >= this.iword().length) {
          throw new Go$Panic("reflect: slice index out of range");
        }
        var typ = this.typ.sliceType.elem;
        var fl = flagAddr | flagIndir | (this.flag & flagRO);
        fl |= typ.Kind() << flagKindShift;
        i += this.iword().offset;
        var array = this.iword().array;
        if (typ.Kind() === go$pkg.Struct) {
          return new Value.Ptr(typ, array[i], fl);
        }
        return new Value.Ptr(typ, new (go$ptrType(typ))(function() { return array[i]; }, function(v) { array[i] = v; }), fl);
      case go$pkg.String:
        var string = this.iword();
        if (i < 0 || i >= string.length) {
          throw new Go$Panic("reflect: string index out of range");
        }
        var fl = (this.flag & flagRO) | (go$pkg.Uint8 << flagKindShift);
        return new Value.Ptr(uint8Type, string.charCodeAt(i), fl);
      }
      throw new Go$Panic(new ValueError.Ptr("reflect.Value.Index", k));
    };
    Value.Ptr.prototype.IsNil = function() {
      switch (this.kind()) {
      case go$pkg.Chan:
      case go$pkg.Func:
      case go$pkg.Ptr:
      case go$pkg.Slice:
        return this.iword() === this.typ.alg.nil;
      case go$pkg.Map:
        return this.iword() === false;
      case go$pkg.Interface:
        return this.iword() === null;
      }
      throw new Go$Panic(new ValueError.Ptr("reflect.Value.IsNil", this.kind()));
    };
    Value.Ptr.prototype.Len = function() {
      var k = this.kind();
      switch (k) {
      case go$pkg.Array:
      case go$pkg.Slice:
      case go$pkg.String:
        return this.iword().length;
      case go$pkg.Map:
        return go$keys(this.iword()).length;
      }
      throw new Go$Panic(new ValueError.Ptr("reflect.Value.Len", k));
    };
    Value.Ptr.prototype.Set = function(x) {
      this.mustBeAssignable();
      x.mustBeExported();
      if ((this.flag & flagIndir) !== 0) {
        switch (this.typ.Kind()) {
        case go$pkg.Interface:
          this.val.go$set(valueInterface(x));
          return;
        case go$pkg.Struct:
          copyStruct(this.val, x.val, this.typ);
          return;
        default:
          this.val.go$set(x.iword());
          return;
        }
      }
      this.val = x.val;
    };
    Value.Ptr.prototype.SetCap = function(n) {
      this.mustBeAssignable();
      this.mustBe(go$pkg.Slice);
      var s = this.val.go$get();
      if (n < s.length || n > s.capacity) {
        throw new Go$Panic("reflect: slice capacity out of range in SetCap");
      }
      var newSlice = new this.typ.alg(s.array);
      newSlice.offset = s.offset;
      newSlice.length = s.length;
      newSlice.capacity = n;
      this.val.go$set(newSlice);
    };
    Value.Ptr.prototype.SetLen = function(n) {
      this.mustBeAssignable();
      this.mustBe(go$pkg.Slice);
      var s = this.val.go$get();
      if (n < 0 || n > s.capacity) {
        throw new Go$Panic("reflect: slice length out of range in SetLen");
      }
      var newSlice = new this.typ.alg(s.array);
      newSlice.offset = s.offset;
      newSlice.length = n;
      newSlice.capacity = s.capacity;
      this.val.go$set(newSlice);
    };
    Value.Ptr.prototype.String = function() {
      switch (this.kind()) {
      case go$pkg.Invalid:
        return "<invalid Value>";
      case go$pkg.String:
        return this.iword();
      }
      return "<" + this.typ.String() + " Value>";
    };
    
    var deepValueEqual = function(a, b, visited) {
      var i;
      if (a === b) {
        return true;
      }
      if (a === null || b === null) {
        return false;
      }
      if (a.constructor === Number) {
        return false;
      }
      if (a.constructor !== b.constructor) {
        return false;
      }
      for (i = 0; i < visited.length; i += 1) {
        var entry = visited[i];
        if (a === entry[0] && b === entry[1]) {
          return true;
        }
      }
      visited.push([a, b]);
      if (a.length !== undefined) {
        if (a.length !== b.length) {
          return false;
        }
        if (a.array !== undefined) {
          for (i = 0; i < a.length; i += 1) {
            if (!deepValueEqual(a.array[a.offset + i], b.array[b.offset + i], visited)) {
              return false;
            }
          }
        } else {
          for (i = 0; i < a.length; i += 1) {
            if (!deepValueEqual(a[i], b[i], visited)) {
              return false;
            }
          }
        }
        return true;
      }
      var keys = go$keys(a), j;
      for (j = 0; j < keys.length; j += 1) {
        var key = keys[j];
        if (key !== "go$id" && key !== "go$val" && !deepValueEqual(a[key], b[key], visited)) {
          return false;
        }
      }
      return true;
    };
    var DeepEqual = function(a, b) {
      return deepValueEqual(a, b, []);
    };
  `
}
