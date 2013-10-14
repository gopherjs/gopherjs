package main

import (
	"angularjs"
	"code.google.com/p/go.tools/go/types"
	"go/build"
	"go/format"
	"go/token"
	"gopherjs/translator"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

type OutputLine struct {
	Type    string
	Content string
}

const gopherjsWebMode = true

func main() {
	app := angularjs.NewModule("playground", nil, nil)

	app.NewController("PlaygroundCtrl", func(scope *angularjs.Scope) {
		scope.Set("code", "package main\n\nimport \"fmt\";\n\nfunc main() {\n\tfmt.Println(\"Hello, playground\")\n}\n")

		var t *translator.Translator
		t = &translator.Translator{
			BuildContext: &build.Context{
				GOROOT:        "/",
				GOOS:          build.Default.GOOS,
				GOARCH:        build.Default.GOARCH,
				Compiler:      "gc",
				InstallSuffix: "js",
				IsDir:         func(name string) bool { return dirs[name] != nil },
				HasSubdir: func(root, dir string) (string, bool) {
					if strings.HasPrefix(dir, root) {
						return dir[len(root):], true
					}
					return "", false
				},
				ReadDir: func(name string) ([]os.FileInfo, error) {
					return dirs[name], nil
				},
				OpenFile: func(name string) (io.ReadCloser, error) {
					if name == "/prog.go" {
						return ioutil.NopCloser(strings.NewReader(scope.GetString("code"))), nil
					}

					content, found := files[name]
					if !found {
						return nil, os.ErrNotExist
					}
					return ioutil.NopCloser(strings.NewReader(content)), nil
				},
			},
			TypesConfig: &types.Config{
				Packages: make(map[string]*types.Package),
			},
			GetModTime: func(name string) time.Time {
				return time.Unix(1, 0)
			},
			StoreArchive: func(pkg *translator.GopherPackage) error {
				return nil
			},
			FileSet:  token.NewFileSet(),
			Packages: make(map[string]*translator.GopherPackage),
		}

		pkg := &translator.GopherPackage{
			Package: &build.Package{
				Name:       "main",
				ImportPath: "main",
				GoFiles:    []string{"prog.go"},
			},
		}

		run := func() {
			err := t.BuildPackage(pkg)
			if err != nil {
				if list, isList := err.(translator.ErrorList); isList {
					output := make([]interface{}, 0)
					for _, entry := range list {
						output = append(output, &OutputLine{"err", entry.Error()})
					}
					scope.Set("output", output)
					return
				}
				scope.Set("output", []interface{}{&OutputLine{"err", err.Error()}})
				return
			}

			scope.Set("output", []interface{}{})
			evalScript(string(pkg.JavaScriptCode), scope)
		}
		scope.Set("run", run)

		scope.Set("format", func() {
			out, err := format.Source([]byte(scope.GetString("code")))
			if err != nil {
				println(err)
				return
			}
			println(out)
		})

		run()
	})
}

func evalScript(script string, scope *angularjs.Scope) {}

const js_evalScript = `
  var Go$webMode = true;
  var console = { log: function() {
  	var lines = Array.prototype.join.call(arguments, " ").split("\n");
  	for (var i = 0; i < lines.length; i++) {
  		scope.native.output.push(new OutputLine("out", lines[i]));
  	}
  } };
  var Go$syscall = function(trap, arg1, arg2, arg3) {
  	switch (trap) {
  	case 4: // SYS_WRITE
  	  var lines = String.fromCharCode.apply(null, arg2).split("\n");
  	  if (scope.native.output.length === 0) {
  	  	scope.native.output.push(new OutputLine("out", ""));
  	  }
  	  scope.native.output[scope.native.output.length - 1].Content += lines[0];
	  	for (var i = 1; i < lines.length; i++) {
	  	  scope.native.output.push(new OutputLine("out", lines[i]));
	  	}
  	  return [arg2.length, 0, null];
  	default:
	  	throw new Go$Panic("Syscall not supported: " + trap);
  	}
  };
  eval(script);
`

type FileEntry struct {
	name string
	mode os.FileMode
}

func (e *FileEntry) Name() string {
	return e.name
}

func (e *FileEntry) Size() int64 {
	return 0
}

func (e *FileEntry) Mode() os.FileMode {
	return e.mode
}

func (e *FileEntry) ModTime() time.Time {
	return time.Unix(1, 0)
}

func (e *FileEntry) IsDir() bool {
	return e.mode.IsDir()
}

func (e *FileEntry) Sys() interface{} {
	return nil
}
