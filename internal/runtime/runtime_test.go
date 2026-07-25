package runtime

import "testing"

func TestParseItems(t *testing.T) {
	items := parseItems("web\tnginx\trunning\napi\tgo\tstopped\n")
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Name != "web" || items[0].Details != "nginx\trunning" {
		t.Fatalf("first item = %+v", items[0])
	}
}

func TestParseKubernetesStyleItems(t *testing.T) {
	items := parseItems("default web Running\nkube-system dns Pending\n")
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Name != "default" || items[0].Details != "web Running" {
		t.Fatalf("first item = %+v", items[0])
	}
}
