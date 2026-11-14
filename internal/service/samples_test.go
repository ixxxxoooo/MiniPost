package service

import (
	"strings"
	"testing"

	"minipost/internal/repository"
)

func newInstaller(t *testing.T) (SampleInstaller, *repository.FileStore) {
	t.Helper()
	store, err := repository.NewFileStoreWithDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return SampleInstaller{
		Projects: NewProjectService(store),
		Requests: NewRequestService(store),
		Envs:     NewEnvironmentService(store),
	}, store
}

func TestSampleProjectsDefinitionsAreValid(t *testing.T) {
	validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true, "OPTIONS": true}
	names := map[string]bool{}
	total := 0
	for _, sp := range SampleProjects() {
		if sp.Name == "" || names[sp.Name] {
			t.Fatalf("项目名为空或重复: %q", sp.Name)
		}
		names[sp.Name] = true
		hasVar := map[string]bool{}
		for _, v := range sp.Variables {
			hasVar[v.Key] = true
		}
		for _, f := range sp.Folders {
			if f.Name == "" || len(f.Requests) == 0 {
				t.Fatalf("%s: 文件夹为空: %q", sp.Name, f.Name)
			}
			for _, r := range f.Requests {
				total++
				if r.Name == "" || !validMethods[strings.ToUpper(r.Method)] || r.URL == "" {
					t.Errorf("%s/%s: 请求定义不完整: %+v", sp.Name, f.Name, r)
				}
				// URL 里用到的变量必须在示例环境里定义
				for _, seg := range strings.Split(r.URL, "{{")[1:] {
					key, _, _ := strings.Cut(seg, "}}")
					if !hasVar[key] {
						t.Errorf("%s/%s: URL 使用了未定义的变量 %q", sp.Name, r.Name, key)
					}
				}
			}
		}
	}
	if len(names) < 4 || total < 40 {
		t.Fatalf("示例应覆盖足够多的场景: %d 个项目 %d 条请求", len(names), total)
	}
}

func TestInstallSamplesCreatesProjectsAndIsIdempotent(t *testing.T) {
	in, store := newInstaller(t)
	created, err := in.InstallSamples()
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != len(SampleProjects()) {
		t.Fatalf("应创建全部示例，实际 %v", created)
	}
	projects, _ := store.ListProjects()
	byName := map[string]string{}
	for _, p := range projects {
		byName[p.Name] = p.ID
	}
	basic := byName["示例 · 请求与认证"]
	if basic == "" {
		t.Fatalf("缺少基础示例项目: %v", byName)
	}
	data, err := store.GetCollectionData(basic)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Folders) != 4 || len(data.Requests) < 15 {
		t.Fatalf("文件夹/请求数量不符: %d / %d", len(data.Folders), len(data.Requests))
	}
	var sawBasic, sawJSON bool
	for _, r := range data.Requests {
		if r.Name == "Basic Auth" && r.Auth.Type == "basic" && r.Auth.Basic.Username == "{{username}}" {
			sawBasic = true
		}
		if r.Name == "POST JSON" && r.Method == "POST" && r.Body.Type == "json" && strings.Contains(r.Body.JSON, "MiniPost") {
			sawJSON = true
		}
		if r.FolderID == "" {
			t.Errorf("请求 %q 应位于文件夹内", r.Name)
		}
	}
	if !sawBasic || !sawJSON {
		t.Fatalf("请求内容未正确保存: basic=%v json=%v", sawBasic, sawJSON)
	}
	envs, _ := store.ListEnvironments(basic)
	if len(envs) != 1 || envs[0].Name != "示例环境" || len(envs[0].Variables) != 4 {
		t.Fatalf("示例环境不符: %+v", envs)
	}
	if p, _ := store.GetProject(basic); p == nil || p.Description == "" {
		t.Fatalf("应写入项目描述")
	}
	// 流式示例项目没有环境，URL 为完整地址
	if envs, _ := store.ListEnvironments(byName["示例 · 流式响应"]); len(envs) != 0 {
		t.Fatalf("流式示例不应创建环境")
	}

	// 再次安装：不重复创建
	again, err := in.InstallSamples()
	if err != nil || len(again) != 0 {
		t.Fatalf("重复安装应跳过，实际 %v %v", again, err)
	}
	if after, _ := store.ListProjects(); len(after) != len(projects) {
		t.Fatalf("项目数量不应变化")
	}
}

func TestInstallSamplesOnlyAddsMissing(t *testing.T) {
	in, store := newInstaller(t)
	first := SampleProjects()[0].Name
	if _, err := in.Projects.CreateProject(first); err != nil {
		t.Fatal(err)
	}
	created, err := in.InstallSamples()
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != len(SampleProjects())-1 {
		t.Fatalf("应只补齐缺少的示例，实际 %v", created)
	}
	projects, _ := store.ListProjects()
	if len(projects) != len(SampleProjects()) {
		t.Fatalf("项目总数不符: %d", len(projects))
	}
}
