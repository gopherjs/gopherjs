package translator

var pkgNatives = make(map[string]map[string]string)

func init() {
	pkgNatives["encoding/gob"] = map[string]string{
		"NewDecoder": `function() { go$notSupported("encoding/gob"); }`,
		"NewEncoder": `function() { go$notSupported("encoding/gob"); }`,
	}

	pkgNatives["math"] = map[string]string{
		"toplevelDependencies": `math:expm1 math:frexp math:hypot math:log10 math:log1p math:log2 math:remainder`,
		"Abs":       `Math.abs`,
		"Acos":      `Math.acos`,
		"Asin":      `Math.asin`,
		"Atan":      `Math.atan`,
		"Atan2":     `Math.atan2`,
		"Ceil":      `Math.ceil`,
		"Copysign":  `function(x, y) { return (x < 0 || 1/x === 1/-0) !== (y < 0 || 1/y === 1/-0) ? -x : x; }`,
		"Cos":       `Math.cos`,
		"Dim":       `function(x, y) { return Math.max(x - y, 0); }`,
		"Exp":       `Math.exp`,
		"Exp2":      `function(x) { return Math.pow(2, x); }`,
		"Expm1":     `function(x) { return expm1(x); }`,
		"Floor":     `Math.floor`,
		"Frexp":     `function(f) { return frexp(f); }`,
		"Hypot":     `function(p, q) { return hypot(p, q); }`,
		"Inf":       `function(sign) { return sign >= 0 ? 1/0 : -1/0; }`,
		"IsInf":     `function(f, sign) { if (f === -1/0) { return sign <= 0; } if (f === 1/0) { return sign >= 0; } return false; }`,
		"IsNaN":     `function(f) { return f !== f; }`,
		"Ldexp":     `go$ldexp`,
		"Log":       `Math.log`,
		"Log10":     `function(x) { return log10(x); }`,
		"Log1p":     `function(x) { return log1p(x); }`,
		"Log2":      `function(x) { return log2(x); }`,
		"Max":       `function(x, y) { return (x === 1/0 || y === 1/0) ? 1/0 : Math.max(x, y); }`,
		"Min":       `function(x, y) { return (x === -1/0 || y === -1/0) ? -1/0 : Math.min(x, y); }`,
		"Mod":       `function(x, y) { return x % y; }`,
		"Modf":      `function(f) { if (f === -1/0 || f === 1/0) { return [f, 0/0]; } var frac = f % 1; return [f - frac, frac]; }`,
		"NaN":       `function() { return 0/0; }`,
		"Pow":       `function(x, y) { return ((x === 1) || (x === -1 && (y === -1/0 || y === 1/0))) ? 1 : Math.pow(x, y); }`,
		"Remainder": `function(x, y) { return remainder(x, y); }`,
		"Signbit":   `function(x) { return x < 0 || 1/x === 1/-0; }`,
		"Sin":       `Math.sin`,
		"Sincos":    `function(x) { return [Math.sin(x), Math.cos(x)]; }`,
		"Sqrt":      `Math.sqrt`,
		"Tan":       `Math.tan`,
		"Trunc":     `function(x) { return (x === 1/0 || x === -1/0 || x !== x || 1/x === 1/-0) ? x : x >> 0; }`,

		// generated from bitcasts/bitcasts.go
		"Float32bits":     `go$float32bits`,
		"Float32frombits": `go$float32frombits`,
		"Float64bits": `function(f) {
			var s, e, x, x$1, x$2, x$3;
			if (f === 0) {
				if (f === 0 && 1 / f === 1 / -0) {
					return new Go$Uint64(2147483648, 0);
				}
				return new Go$Uint64(0, 0);
			}
			if (f !== f) {
				return new Go$Uint64(2146959360, 1);
			}
			s = new Go$Uint64(0, 0);
			if (f < 0) {
				s = new Go$Uint64(2147483648, 0);
				f = -f;
			}
			e = 1075;
			while (f >= 9.007199254740992e+15) {
				f = f / 2;
				if (e === 2047) {
					break;
				}
				e = e + 1 >>> 0;
			}
			while (f < 4.503599627370496e+15) {
				e = e - 1 >>> 0;
				if (e === 0) {
					break;
				}
				f = f * 2;
			}
			return (x = (x$1 = go$shiftLeft64(new Go$Uint64(0, e), 52), new Go$Uint64(s.high | x$1.high, (s.low | x$1.low) >>> 0)), x$2 = (x$3 = new Go$Uint64(0, f), new Go$Uint64(x$3.high &~ 1048576, (x$3.low &~ 0) >>> 0)), new Go$Uint64(x.high | x$2.high, (x.low | x$2.low) >>> 0));
		}`,
		"Float64frombits": `function(b) {
			var s, x, x$1, e, m;
			s = 1;
			if (!((x = new Go$Uint64(b.high & 2147483648, (b.low & 0) >>> 0), (x.high === 0 && x.low === 0)))) {
				s = -1;
			}
			e = (x$1 = go$shiftRightUint64(b, 52), new Go$Uint64(x$1.high & 0, (x$1.low & 2047) >>> 0));
			m = new Go$Uint64(b.high & 1048575, (b.low & 4294967295) >>> 0);
			if ((e.high === 0 && e.low === 2047)) {
				if ((m.high === 0 && m.low === 0)) {
					return s / 0;
				}
				return 0/0;
			}
			if (!((e.high === 0 && e.low === 0))) {
				m = new Go$Uint64(m.high + 1048576, m.low + 0);
			}
			if ((e.high === 0 && e.low === 0)) {
				e = new Go$Uint64(0, 1);
			}
			return go$ldexp(go$flatten64(m), ((e.low >> 0) - 1023 >> 0) - 52 >> 0) * s;
		}`,
	}

	pkgNatives["os"] = map[string]string{
		"toplevel": `
			if (go$packages["syscall"].Syscall15 !== undefined) { // windows
				NewFile = go$pkg.NewFile = function() { return new File.Ptr(); };
			}
		`,
		"toplevelDependencies": `syscall:Syscall15 os:NewFile`,
		"Args":                 `new (go$sliceType(Go$String))((typeof process !== 'undefined') ? process.argv.slice(1) : [])`,
	}

	pkgNatives["runtime"] = map[string]string{
		"toplevel": `
			go$throwRuntimeError = function(msg) { throw go$panic(new errorString(msg)); };
		`,
		"toplevelDependencies": `runtime:errorString runtime:TypeAssertionError`,
		"getgoroot": `function() {
			return (typeof process !== 'undefined') ? (process.env["GOROOT"] || "") : "/";
		}`,
		"sizeof_C_MStats": `3712`,
		"Caller": `function(skip) {
			var line = go$getStack()[skip + 3];
			if (line === undefined) {
				return [0, "", 0, false];
			}
			var parts = line.substring(line.indexOf("(") + 1, line.indexOf(")")).split(":");
			return [0, parts[0], parseInt(parts[1]), true];
		}`,
		"GC": `function() {}`,
		"GOMAXPROCS": `function(n) {
			if (n > 1) {
				go$notSupported("GOMAXPROCS != 1");
			}
			return 1;
		}`,
		"Goexit": `function() {
			var err = new Go$Error();
			err.go$exit = true;
			throw err;
		}`,
		"NumCPU":       `function() { return 1; }`,
		"ReadMemStats": `function() {}`,
		"SetFinalizer": `function() {}`,
	}

	pkgNatives["sync"] = map[string]string{
		"copyChecker.check":    `function() {}`,
		"runtime_Syncsemcheck": `function() {}`,
	}

	pkgNatives["syscall"] = map[string]string{
		"toplevel": `
			if (go$pkg.Syscall15 !== undefined) { // windows
				Syscall = Syscall6 = Syscall9 = Syscall12 = Syscall15 = go$pkg.Syscall = go$pkg.Syscall6 = go$pkg.Syscall9 = go$pkg.Syscall12 = go$pkg.Syscall15 = loadlibrary = getprocaddress = function() { throw new Error("Syscalls not available."); };
				getStdHandle = GetCommandLine = go$pkg.GetCommandLine = function() {};
				CommandLineToArgv = go$pkg.CommandLineToArgv = function() { return [null, {}]; };
				Getenv = go$pkg.Getenv = function(key) { return ["", false]; };
				GetTimeZoneInformation = go$pkg.GetTimeZoneInformation = function() { return [undefined, true]; };
			} else if (typeof process === "undefined") {
				var syscall = function() { throw new Error("Syscalls not available."); };
				if (typeof go$syscall !== "undefined") {
					syscall = go$syscall;
				}
				Syscall = Syscall6 = RawSyscall = RawSyscall6 = go$pkg.Syscall = go$pkg.Syscall6 = go$pkg.RawSyscall = go$pkg.RawSyscall6 = syscall;
				envs = new (go$sliceType(Go$String))(new Array(0));
			} else {
				try {
					var syscall = require("syscall");
					Syscall = go$pkg.Syscall = syscall.Syscall;
					Syscall6 = go$pkg.Syscall6 = syscall.Syscall6;
					RawSyscall = go$pkg.RawSyscall = syscall.Syscall;
					RawSyscall6 = go$pkg.RawSyscall6 = syscall.Syscall6;
				} catch (e) {
					Syscall = Syscall6 = RawSyscall = RawSyscall6 = go$pkg.Syscall = go$pkg.Syscall6 = go$pkg.RawSyscall = go$pkg.RawSyscall6 = function() { throw e; };
				}
				BytePtrFromString = go$pkg.BytePtrFromString = function(s) { return [go$stringToBytes(s, true), null]; };

				var envkeys = Object.keys(process.env);
				envs = new (go$sliceType(Go$String))(new Array(envkeys.length));
				var i;
				for(i = 0; i < envkeys.length; i++) {
					envs.array[i] = envkeys[i] + "=" + process.env[envkeys[i]];
				}
			}
		`,
		"toplevelDependencies": `syscall:Syscall syscall:Syscall6 syscall:Syscall9 syscall:Syscall12 syscall:Syscall15 syscall:loadlibrary syscall:getprocaddress syscall:getStdHandle syscall:GetCommandLine syscall:CommandLineToArgv syscall:Getenv syscall:GetTimeZoneInformation syscall:RawSyscall syscall:RawSyscall6 syscall:BytePtrFromString`,
	}

	pkgNatives["testing"] = map[string]string{
		"runTest": `function(f, t) {
			try {
				f(t);
				return null;
			} catch (e) {
				go$jsErr = null;
				return e;
			}
		}`,
	}

	pkgNatives["time"] = map[string]string{
		"now":   `go$now`,
		"After": `function() { go$notSupported("time.After (use time.AfterFunc instead)") }`,
		"AfterFunc": `function(d, f) {
			setTimeout(f, go$div64(d, new Duration(0, 1000000)).low);
			return null;
		}`,
		"NewTimer": `function() { go$notSupported("time.NewTimer (use time.AfterFunc instead)") }`,
		"Sleep":    `function() { go$notSupported("time.Sleep (use time.AfterFunc instead)") }`,
		"Tick":     `function() { go$notSupported("time.Tick (use time.AfterFunc instead)") }`,
	}

	pkgNatives["github.com/gopherjs/gopherjs/js"] = map[string]string{
		"toplevelDependencies": `github.com/gopherjs/gopherjs/js:Error`,
	}

	pkgNatives["github.com/gopherjs/gopherjs/js_test"] = map[string]string{
		"dummys": `{
			someBool: true,
			someString: "abc\u1234",
			someInt: 42,
			someFloat: 42.123,
			someArray: [41, 42, 43],
			add: function(a, b) {
				return a + b;
			},
			mapArray: go$mapArray,
			toUnixTimestamp: function(d) {
				return d.getTime() / 1000;
			},
		}`,
	}
}
