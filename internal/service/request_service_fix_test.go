package service

import "testing"

// TestDuplicateFolder_CopiesNestedSubtree 验证复制文件夹会递归复制子文件夹及其中的请求。
func TestDuplicateFolder_CopiesNestedSubtree(t *testing.T) {
	svc, _, projectID := newImportTestService(t)

	parent, err := svc.CreateFolder(projectID, "", "Parent")
	if err != nil {
		t.Fatalf("创建父文件夹失败: %v", err)
	}
	child, err := svc.CreateFolder(projectID, parent.ID, "Child")
	if err != nil {
		t.Fatalf("创建子文件夹失败: %v", err)
	}
	if _, err := svc.CreateRequest(projectID, parent.ID, "DirectReq"); err != nil {
		t.Fatalf("创建直接子请求失败: %v", err)
	}
	if _, err := svc.CreateRequest(projectID, child.ID, "NestedReq"); err != nil {
		t.Fatalf("创建嵌套请求失败: %v", err)
	}

	dup, err := svc.DuplicateFolder(projectID, parent.ID)
	if err != nil {
		t.Fatalf("复制文件夹失败: %v", err)
	}
	if dup.Name != "Parent (副本)" {
		t.Fatalf("副本名称不符合预期: %q", dup.Name)
	}

	data, err := svc.GetCollectionData(projectID)
	if err != nil {
		t.Fatalf("读取集合数据失败: %v", err)
	}

	// 副本根文件夹下应有一个子文件夹（Child 的副本）
	var childCopyID string
	childCopyCount := 0
	for _, f := range data.Folders {
		if f.ParentID == dup.ID {
			childCopyCount++
			childCopyID = f.ID
			if f.Name != "Child" {
				t.Fatalf("嵌套子文件夹名称不符合预期: %q", f.Name)
			}
		}
	}
	if childCopyCount != 1 {
		t.Fatalf("副本应包含 1 个嵌套子文件夹, 实际 %d", childCopyCount)
	}

	directCount := 0
	nestedCount := 0
	for _, r := range data.Requests {
		switch r.FolderID {
		case dup.ID:
			directCount++
		case childCopyID:
			nestedCount++
		}
	}
	if directCount != 1 {
		t.Fatalf("副本直接子请求应为 1, 实际 %d", directCount)
	}
	if nestedCount != 1 {
		t.Fatalf("副本嵌套请求应为 1, 实际 %d", nestedCount)
	}

	// 文件夹总数：原 Parent + 原 Child + 副本 Parent + 副本 Child = 4
	if len(data.Folders) != 4 {
		t.Fatalf("复制后文件夹总数应为 4, 实际 %d", len(data.Folders))
	}
}

func TestRenameFolder_NotFoundReturnsError(t *testing.T) {
	svc, _, projectID := newImportTestService(t)
	if err := svc.RenameFolder(projectID, "missing-folder", "X"); err == nil {
		t.Fatal("重命名不存在的文件夹应返回错误")
	}
}

func TestRenameRequest_NotFoundReturnsError(t *testing.T) {
	svc, _, projectID := newImportTestService(t)
	if err := svc.RenameRequest(projectID, "missing-request", "X"); err == nil {
		t.Fatal("重命名不存在的请求应返回错误")
	}
}
