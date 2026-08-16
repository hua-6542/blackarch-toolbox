package decision

import (
	"errors"
	"testing"

	"blackarch-toolbox/internal/model"
)

type fakePrefs struct {
	env string
	ok  bool
	err error
}

func (f fakePrefs) GetPreference(int64) (string, bool, error) { return f.env, f.ok, f.err }

func engineWith(prefs PreferenceStore, which map[string]string) *Engine {
	e := New(prefs)
	e.Which = func(bin string) (string, bool) {
		p, ok := which[bin]
		return p, ok
	}
	return e
}

func TestDecideRequestedEnv(t *testing.T) {
	e := engineWith(fakePrefs{}, nil)
	d, err := e.Decide(model.Tool{Name: "nmap"}, "podman")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env != "podman" || d.Priority != 1 {
		t.Fatalf("显式指定环境失败: %+v", d)
	}
}

func TestDecidePreferenceWins(t *testing.T) {
	e := engineWith(fakePrefs{env: "vm", ok: true}, map[string]string{"nmap": "/usr/bin/nmap"})
	d, err := e.Decide(model.Tool{ID: 1, Name: "nmap"}, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env != "vm" || d.Priority != 1 {
		t.Fatalf("偏好应优先: %+v", d)
	}
}

func TestDecideHighRisk(t *testing.T) {
	e := engineWith(fakePrefs{}, map[string]string{"metasploit": "/usr/bin/metasploit"})
	d, err := e.Decide(model.Tool{ID: 2, Name: "metasploit", IsHighRisk: true}, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env != "vm" || d.Priority != 2 {
		t.Fatalf("高危应走 vm: %+v", d)
	}
}

func TestDecideDependencyConflict(t *testing.T) {
	e := engineWith(fakePrefs{}, map[string]string{"beef-xss": "/usr/bin/beef-xss"})
	d, err := e.Decide(model.Tool{ID: 3, Name: "beef-xss", Dependencies: []string{"python2"}}, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env != "podman" || d.Priority != 3 {
		t.Fatalf("依赖冲突应走 podman: %+v", d)
	}
}

func TestDecideLocalExists(t *testing.T) {
	e := engineWith(fakePrefs{}, map[string]string{"nmap": "/usr/bin/nmap"})
	d, err := e.Decide(model.Tool{ID: 4, Name: "nmap"}, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env != "local" || d.Priority != 4 {
		t.Fatalf("本地存在应走 local: %+v", d)
	}
}

func TestDecideFallbackVM(t *testing.T) {
	e := engineWith(fakePrefs{}, nil)
	d, err := e.Decide(model.Tool{ID: 5, Name: "不存在工具"}, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env != "vm" || d.Priority != 5 {
		t.Fatalf("兜底应走 vm: %+v", d)
	}
}

func TestDecideInvalidEnv(t *testing.T) {
	e := engineWith(fakePrefs{}, nil)
	if _, err := e.Decide(model.Tool{Name: "nmap"}, "docker"); err == nil {
		t.Fatal("非法环境应报错")
	}
}

func TestDecidePrefError(t *testing.T) {
	e := engineWith(fakePrefs{err: errors.New("boom")}, nil)
	if _, err := e.Decide(model.Tool{ID: 9, Name: "nmap"}, "auto"); err == nil {
		t.Fatal("偏好查询错误应上抛")
	}
}

func TestHighRiskListContainsRequired(t *testing.T) {
	want := []string{"metasploit", "volatility", "reaver", "bully", "bettercap", "aircrack-ng", "mdk3", "mdk4"}
	got := make(map[string]bool)
	for _, v := range HighRiskTools {
		got[v] = true
	}
	for _, w := range want {
		if !got[w] {
			t.Fatalf("HighRiskTools 缺 %s", w)
		}
	}
}
