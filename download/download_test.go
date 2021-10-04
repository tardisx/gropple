package download

import "testing"

func TestUpdateMetadata(t *testing.T) {
	newD := Download{}

	// first time we spot a filename
	newD.updateMetadata("[download] Destination: Halo Infinite Flight 4K Gameplay-wi7Agv1M6PY.f401.mp4")
	if len(newD.Files) != 1 || newD.Files[0] != "Halo Infinite Flight 4K Gameplay-wi7Agv1M6PY.f401.mp4" {
		t.Fatalf("incorrect Files:%v", newD.Files)
	}

	// eta's might be xx:xx:xx or xx:xx
	newD.updateMetadata("[download]   0.0% of 504.09MiB at 135.71KiB/s ETA 01:03:36")
	if newD.Eta != "01:03:36" {
		t.Fatalf("bad long eta in dl\n%v", newD)
	}
	newD.updateMetadata("[download]   0.0% of 504.09MiB at 397.98KiB/s ETA 21:38")
	if newD.Eta != "21:38" {
		t.Fatalf("bad short eta in dl\n%v", newD)
	}

	// added a new file, now we are tracking two
	newD.updateMetadata("[download] Destination: Halo Infinite Flight 4K Gameplay-wi7Agv1M6PY.f140.m4a")
	if len(newD.Files) != 2 || newD.Files[1] != "Halo Infinite Flight 4K Gameplay-wi7Agv1M6PY.f140.m4a" {
		t.Fatalf("incorrect Files:%v", newD.Files)
	}

	// merging
	newD.updateMetadata("[ffmpeg] Merging formats into \"Halo Infinite Flight 4K Gameplay-wi7Agv1M6PY.mp4\"")
	if len(newD.Files) != 3 || newD.Files[2] != "Halo Infinite Flight 4K Gameplay-wi7Agv1M6PY.mp4" {
		t.Fatalf("did not find merged filename")
		t.Fatalf("%v", newD.Files)
	}

	// deletes
	// TODO. Not sure why I don't always see the "Deleting original file" messages after merge -
	// maybe a youtube-dl fork thing?

}

// [youtube] wi7Agv1M6PY: Downloading webpage
// [info] Writing video description metadata as JSON to: Halo Infinite Flight 4K Gameplay-wi7Agv1M6PY.info.json
// [download] Destination: Halo Infinite Flight 4K Gameplay-wi7Agv1M6PY.f401.mp4
// [download]   0.0% of 504.09MiB at 135.71KiB/s ETA 01:03:36
// [download]   0.0% of 504.09MiB at 397.98KiB/s ETA 21:38
// [download]   0.0% of 504.09MiB at 918.97KiB/s ETA 09:22
// [download]   0.0% of 504.09MiB at  1.90MiB/s ETA 04:25
// ..
// [download]  99.6% of 504.09MiB at  8.91MiB/s ETA 00:00
// [download] 100.0% of 504.09MiB at  9.54MiB/s ETA 00:00
// [download] 100% of 504.09MiB in 01:00
// [download] Destination: Halo Infinite Flight 4K Gameplay-wi7Agv1M6PY.f140.m4a
// [download]   0.0% of 4.64MiB at 155.26KiB/s ETA 00:30
// [download]   0.1% of 4.64MiB at 457.64KiB/s ETA 00:10
// [download]   0.1% of 4.64MiB at  1.03MiB/s ETA 00:04
// ..
// [download]  86.2% of 4.64MiB at 10.09MiB/s ETA 00:00
// [download] 100.0% of 4.64MiB at 10.12MiB/s ETA 00:00
// [download] 100% of 4.64MiB in 00:00
// [ffmpeg] Merging formats into "Halo Infinite Flight 4K Gameplay-wi7Agv1M6PY.mp4"
