package handler

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestExportMonitorFromModelDefaultsAcceptedCodes(t *testing.T) {
	exported := exportMonitorFromModel(model.Monitor{
		Name:                "site",
		Type:                model.MonitorTypeHTTP,
		AcceptedStatusCodes: "",
	}, []model.Tag{{Name: "prod", Color: "#123456"}}, []string{"ops"}, []string{"root"})

	if len(exported.AcceptedStatusCodes) != 1 || exported.AcceptedStatusCodes[0] != "200-299" {
		t.Fatalf("accepted codes=%v", exported.AcceptedStatusCodes)
	}
	if len(exported.Tags) != 1 || exported.Tags[0].Name != "prod" || exported.Tags[0].Color != "#123456" {
		t.Fatalf("tags=%+v", exported.Tags)
	}
	if len(exported.NotificationNames) != 1 || exported.NotificationNames[0] != "ops" {
		t.Fatalf("notifications=%+v", exported.NotificationNames)
	}
	if len(exported.GroupPath) != 1 || exported.GroupPath[0] != "root" {
		t.Fatalf("group_path=%+v", exported.GroupPath)
	}
}

func TestBuildImportPreviewSummarizesConflictsTagsAndMaskedNotifications(t *testing.T) {
	preview := buildImportPreview(map[string]model.Monitor{
		"existing": {ID: 9, Name: "existing", Type: model.MonitorTypeHTTP},
	}, ExportFile{
		Monitors: []ExportMonitor{
			{Name: "existing", Type: model.MonitorTypeHTTP},
			{Name: "new-a", Type: model.MonitorTypeHTTP, Tags: []ExportTag{{Name: "prod", Color: "#111111"}}},
			{Name: "new-b", Type: model.MonitorTypeTCP, Tags: []ExportTag{{Name: "prod", Color: "#222222"}}},
		},
		Notifications: []ExportNotification{
			{Name: "ops", Config: `{"webhook_url":"***"}`},
			{Name: ""},
		},
	})

	if preview.ConflictCount != 1 || preview.NewCount != 2 {
		t.Fatalf("counts conflict=%d new=%d", preview.ConflictCount, preview.NewCount)
	}
	if len(preview.Conflicts) != 1 || preview.Conflicts[0].ExistingID != 9 {
		t.Fatalf("conflicts=%+v", preview.Conflicts)
	}
	if len(preview.NewTags) != 1 || preview.NewTags[0].Name != "prod" {
		t.Fatalf("new tags=%+v", preview.NewTags)
	}
	if preview.Notifications != 1 || preview.MaskedNotifications != 1 {
		t.Fatalf("notification counts=%d masked=%d", preview.Notifications, preview.MaskedNotifications)
	}
	if preview.Summary == "" {
		t.Fatal("summary missing")
	}
}
