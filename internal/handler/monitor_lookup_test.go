package handler

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestUserMonitorLooksUpOwnedMonitorOnly(t *testing.T) {
	db := testDB(t)
	owned := model.Monitor{UserID: 1, Name: "owned", Type: model.MonitorTypeHTTP}
	other := model.Monitor{UserID: 2, Name: "other", Type: model.MonitorTypeHTTP}
	if err := db.Create(&owned).Error; err != nil {
		t.Fatalf("create owned monitor: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other monitor: %v", err)
	}

	got, err := userMonitor(db, 1, owned.ID)
	if err != nil {
		t.Fatalf("userMonitor owned: %v", err)
	}
	if got.ID != owned.ID {
		t.Fatalf("got monitor id=%d want %d", got.ID, owned.ID)
	}
	if _, err := userMonitor(db, 1, other.ID); err == nil {
		t.Fatalf("expected cross-user monitor lookup to fail")
	}
}

func TestUserGroupMonitorRequiresGroupType(t *testing.T) {
	db := testDB(t)
	group := model.Monitor{UserID: 1, Name: "group", Type: model.MonitorTypeGroup}
	leaf := model.Monitor{UserID: 1, Name: "leaf", Type: model.MonitorTypeHTTP}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group monitor: %v", err)
	}
	if err := db.Create(&leaf).Error; err != nil {
		t.Fatalf("create leaf monitor: %v", err)
	}

	got, err := userGroupMonitor(db, 1, group.ID)
	if err != nil {
		t.Fatalf("userGroupMonitor group: %v", err)
	}
	if got.ID != group.ID {
		t.Fatalf("got group id=%d want %d", got.ID, group.ID)
	}
	if _, err := userGroupMonitor(db, 1, leaf.ID); err == nil {
		t.Fatalf("expected non-group monitor lookup to fail")
	}
}

func TestUserGroupPathBuildsOwnedAncestorNames(t *testing.T) {
	db := testDB(t)
	root := model.Monitor{UserID: 1, Name: "root", Type: model.MonitorTypeGroup}
	if err := db.Create(&root).Error; err != nil {
		t.Fatalf("create root: %v", err)
	}
	child := model.Monitor{UserID: 1, Name: "child", Type: model.MonitorTypeGroup, GroupID: &root.ID}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	other := model.Monitor{UserID: 2, Name: "other", Type: model.MonitorTypeGroup}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other: %v", err)
	}

	path := userGroupPath(db, 1, &child.ID)
	if len(path) != 2 || path[0] != "root" || path[1] != "child" {
		t.Fatalf("path=%v", path)
	}
	if path := userGroupPath(db, 1, nil); path != nil {
		t.Fatalf("nil group path=%v", path)
	}
	if path := userGroupPath(db, 1, &other.ID); len(path) != 0 {
		t.Fatalf("cross-user path=%v", path)
	}
}

func TestWouldCreateGroupCycleFollowsOwnedParentChain(t *testing.T) {
	db := testDB(t)
	root := model.Monitor{UserID: 1, Name: "root", Type: model.MonitorTypeGroup}
	if err := db.Create(&root).Error; err != nil {
		t.Fatalf("create root: %v", err)
	}
	child := model.Monitor{UserID: 1, Name: "child", Type: model.MonitorTypeGroup, GroupID: &root.ID}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	leaf := model.Monitor{UserID: 1, Name: "leaf", Type: model.MonitorTypeGroup, GroupID: &child.ID}
	if err := db.Create(&leaf).Error; err != nil {
		t.Fatalf("create leaf: %v", err)
	}

	cycle, err := wouldCreateGroupCycle(db, 1, root.ID, leaf.ID)
	if err != nil {
		t.Fatalf("cycle check: %v", err)
	}
	if !cycle {
		t.Fatalf("expected assigning root under leaf to create a cycle")
	}
	cycle, err = wouldCreateGroupCycle(db, 1, leaf.ID, root.ID)
	if err != nil {
		t.Fatalf("cycle check: %v", err)
	}
	if cycle {
		t.Fatalf("did not expect assigning leaf under root to create a cycle")
	}
}

func TestWouldCreateGroupCycleReturnsLookupErrors(t *testing.T) {
	db := testDB(t)
	root := model.Monitor{UserID: 1, Name: "root", Type: model.MonitorTypeGroup}
	if err := db.Create(&root).Error; err != nil {
		t.Fatalf("create root: %v", err)
	}
	if err := db.Migrator().DropTable(&model.Monitor{}); err != nil {
		t.Fatalf("drop monitors: %v", err)
	}

	if _, err := wouldCreateGroupCycle(db, 1, root.ID, root.ID+1); err == nil {
		t.Fatal("expected lookup error")
	}
}
