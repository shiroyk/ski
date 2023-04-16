package jsmodule

import (
	"testing"
)

type testModule struct{}

func (t testModule) Exports() any { return map[string]string{"foo": "module"} }

type testGlobalModule struct{}

func (t testGlobalModule) Exports() any { return map[string]string{"foo": "global"} }
func (t testGlobalModule) Global()      {}

func TestModule(t *testing.T) {
	t.Parallel()

	moduleKey := "testModule"
	if _, ok := GetModule(ExtPrefix + moduleKey); !ok {
		Register(moduleKey, new(testModule))
	}
	if _, ok := GetModule(ExtPrefix + moduleKey); !ok {
		t.Fatal("unable get module")
	}

	globalModuleKey := "testModule"
	if _, ok := GetModule(globalModuleKey); !ok {
		Register(globalModuleKey, new(testGlobalModule))
	}
	if _, ok := GetModule(globalModuleKey); !ok {
		t.Fatal("unable get global module")
	}
}
