package dom

import (
	"errors"
	"github.com/psilva261/sparkle/console"
	"github.com/psilva261/sparkle/eventloop"
	"github.com/psilva261/sparkle/js"
	"github.com/psilva261/sparkle/require"
	"github.com/psilva261/sparklefs/logger"
	"os"
	"path/filepath"
	"syscall"
)

func W3C(testFn string) (err error) {
	l := eventloop.NewEventLoop()
	l.Start()
	defer l.Stop()
	l.RunOnLoop(func(vm *js.Runtime) {
		registry := require.NewRegistry(
			require.WithLoader(
				require.SourceLoader(w3cSrcLoader),
			),
		)
		console.Enable(vm)
		registry.Enable(vm)
		bs, err := os.ReadFile("test/w3c/harness/DomTestCase.js")
		if err != nil {
			log.Fatalf(err.Error())
		}
		_, err = vm.RunString(string(bs))
		if err != nil {
			log.Fatalf(err.Error())
		}
		bs, err = os.ReadFile("test/w3c/level1/core/" + testFn)
		if err != nil {
			log.Fatalf(err.Error())
		}
		_, err = vm.RunString(string(bs))
		if err != nil {
			log.Fatalf(err.Error())
		}
		vm.Set("readFileSync", vm.ToValue(func(fn string) (v js.Value) {
			bs, err := os.ReadFile("test/w3c/level1/core/" + fn)
			if err != nil {
				log.Fatalf(err.Error())
			}
			return vm.ToValue(string(bs))
		}))
		vm.Set("initTheDocument", vm.ToValue(func(htm string) (o *js.Object) {
			d, err := Init(vm, htm, "")
			if err != nil {
				log.Fatalf(err.Error())
			}
			return d.Obj()
		}))
		vm.Set("assertEquals", vm.ToValue(func(msg string, exp, act js.Value) {
			//_ = exp.(js.Value)
			//_ = act.(js.Value)
			if !exp.Equals(act) {
				log.Printf("%v", msg)
				log.Fatalf("%v (exp. %v but got %v)", msg, exp, act)
			}
		}))
		vm.Set("assertNotEqual", vm.ToValue(func(msg string, exp, act js.Value) {
			//_ = exp.(js.Value)
			//_ = act.(js.Value)
			if exp.Equals(act) {
				log.Printf("%v", msg)
				log.Fatalf("%v (exp. %v but did not want to get %v)", msg, exp, act)
			}
		}))
		_, err = vm.RunString(`
alert = console.log;
__dirname = '';
Path = {
	resolve: function() {

	}
};
assertTrue = function(message, actual) {
	assertEquals(message, true, actual);
};
assertFalse = function(message, actual) {
	assertEquals(message, true, !actual);
},
assertNull = function(message, actual) {
	assertEquals(message, null, actual);
};
assertNotNull = function(message, actual) {
	assertNotEqual(actual, null, message);
};
impl = {
	hasFeature: function(feature, version) {
		console.log('hasFeature(' + feature + ', ' + version + ')');
		if (feature == 'XML') {
			return false;
		}
		return true;
	}
};
createConfiguredBuilder = function() {
  return {
    contentType: 'text/html',
    hasFeature: function(feature, version) {
      return impl.hasFeature(feature, version);
    },
    getImplementation: function() {
      return impl;
    },
    setImplementationAttribute: function(attr, value) {
      // Ignore
    },
    preload: function(docRef, name, href) {
      return 1;
    },
    load: function(docRef, name, href) {
      var doc = 'files/' + href + '.html';
      var html = readFileSync(doc);
      var url = 'http://example.com/'+['files',href].join('/')+'.html';
      return initTheDocument(html);
    }
  };
};
		`)
		if err != nil {
			log.Fatalf(err.Error())
		}
		_, err = vm.RunString(`setUpPage()`)
		if err != nil {
			log.Fatalf(err.Error())
		}
		_, err = vm.RunString(`runTest()`)
		if err != nil {
			log.Fatalf(err.Error())
		}
	})
	return
}

func w3cSrcLoader(fn string) ([]byte, error) {
	path := filepath.FromSlash("../test/w3c/" + fn)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, syscall.EISDIR) {
			log.Printf("fvffd2 %v", path)
			err = require.ModuleFileDoesNotExistError
		} else {
			log.Printf("srcLoader: handling of require('%v') is not implemented", fn)
		}
	}
	return data, err
}
