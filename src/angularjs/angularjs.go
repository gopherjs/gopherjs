package angularjs

func NewModule(name string, requires []string, configFn func()) *Module { return nil }

const js_NewModule = `
	requires = requires ? requires.Go$toArray() : [];
	return new Module(angular.module(name, requires, configFn));
`

type Module struct {
	native interface{}
	SCE    *SCE
}

func (m *Module) NewController(name string, constructor func(scope *Scope)) {}

const js_Module_NewController = `
	this.native.controller(name, function($scope, $sce) {
		constructor(new Scope($scope, new SCE($sce)));
	});
`

func (m *Module) NewFilter(name string, fn func(text string, arguments []string) string) {}

const js_Module_NewFilter = `
	this.native.filter(name, function() {
		return function(text) {
			return fn(text, new Go$Slice(Array.prototype.slice.call(arguments, 1)));
		};
	});
`

type Scope struct {
	native interface{}
}

func (s *Scope) GetString(key string) string { return "" }

const js_Scope_GetString = `
	return Go$internalizeString(String(this.native[key]));
`

func (s *Scope) GetInt(key string) int { return 0 }

const js_Scope_GetInt = `
	return parseInt(this.native[key]);
`

func (s *Scope) GetFloat(key string) float64 { return 0 }

const js_Scope_GetFloat = `
	return parseFloat(this.native[key]);
`

func (s *Scope) GetSlice(key string) []interface{} { return nil }

const js_Scope_GetSlice = `
	return new Go$Slice(this.native[key]);
`

func (s *Scope) Set(key string, value interface{}) {}

const js_Scope_Set = `
	switch (value.constructor) {
	case Go$String:
		this.native[key] = Go$externalizeString(value.v);
		break;
	case Go$Slice:
		this.native[key] = value.Go$toArray();
		break;
	default:
		this.native[key] = value.v !== undefined ? value.v : value;
		break;
	}
`

type SCE struct {
	native interface{}
}
