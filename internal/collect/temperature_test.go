package collect

import "testing"

func TestFriendlySensorLabel(t *testing.T) {
	cases := map[string]string{
		"k10temp_tctl":          "CPU",
		"coretemp_package_id_0": "CPU",
		"amdgpu_edge":           "Integrated Graphics",
		"nvme_composite":        "NVMe SSD",
		"nvme_sensor_1":         "NVMe SSD",
		"acpitz":                "Motherboard",
		"mt7921_phy0":           "Wi-Fi",
		"iwlwifi_1":             "Wi-Fi",
		"something_unmapped":    "something_unmapped",
	}
	for key, want := range cases {
		if got := friendlySensorLabel(key); got != want {
			t.Errorf("friendlySensorLabel(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestRedundantSensor(t *testing.T) {
	if !redundantSensor("nvme_sensor_1") {
		t.Error("per-die nvme probes should be dropped")
	}
	if redundantSensor("nvme_composite") {
		t.Error("nvme composite is the headline reading and must be kept")
	}
}

func TestNumberDuplicates(t *testing.T) {
	sensors := []Sensor{
		{Label: "CPU", Detail: "k10temp_tctl"},
		{Label: "NVMe SSD", Detail: "nvme_composite"},
		{Label: "NVMe SSD", Detail: "nvme_composite"},
		{Label: "NVMe SSD", Detail: "nvme_sensor_1"},
	}
	numberDuplicates(sensors)

	want := []string{"CPU", "NVMe SSD 1", "NVMe SSD 2", "NVMe SSD 3"}
	for i, label := range want {
		if sensors[i].Label != label {
			t.Errorf("sensors[%d].Label = %q, want %q", i, sensors[i].Label, label)
		}
	}
}
