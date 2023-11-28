package download

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tardisx/gropple/config"
)

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
		t.Fatalf("bad long eta in dl\n%#v", newD)
	}
	newD.updateMetadata("[download]   0.0% of 504.09MiB at 397.98KiB/s ETA 21:38")
	if newD.Eta != "21:38" {
		t.Fatalf("bad short eta in dl\n%#v", newD)
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

	// different download
	newD.updateMetadata("[download]  99.3% of ~1.42GiB at 320.87KiB/s ETA 00:07 (frag 212/214)")
	if newD.Eta != "00:07" {
		t.Fatalf("bad short eta in dl with frag\n%v", newD)
	}

	// [FixupM3u8] Fixing MPEG-TS in MP4 container of "file [-168849776_456239489].mp4"
	newD.updateMetadata("[FixupM3u8] Fixing MPEG-TS in MP4 container of \"file [-168849776_456239489].mp4")
	if newD.State != "Fixing MPEG-TS in MP4" {
		t.Fatalf("did not see fixup state - state is %s", newD.State)
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

func TestQueue(t *testing.T) {
	cs := config.ConfigService{}
	cs.LoadTestConfig()
	conf := cs.Config

	new1 := NewDownload("http://sub.example.org/foo1", conf)
	new2 := NewDownload("http://sub.example.org/foo2", conf)
	new3 := NewDownload("http://sub.example.org/foo3", conf)
	new4 := NewDownload("http://example.org/", conf)

	// pretend the user chose a profile for each
	new1.DownloadProfile = *conf.ProfileCalled("test profile")
	new2.DownloadProfile = *conf.ProfileCalled("test profile")
	new3.DownloadProfile = *conf.ProfileCalled("test profile")
	new4.DownloadProfile = *conf.ProfileCalled("test profile")
	new1.State = STATE_QUEUED
	new2.State = STATE_QUEUED
	new3.State = STATE_QUEUED
	new4.State = STATE_QUEUED

	q := Manager{
		Downloads:    []*Download{},
		MaxPerDomain: 2,
		Lock:         sync.Mutex{},
	}

	q.AddDownload(new1)
	q.AddDownload(new2)
	q.AddDownload(new3)
	q.AddDownload(new4)

	q.startQueued(1)

	// two should start, one from each of the two domains
	time.Sleep(time.Millisecond * 100)
	if q.Downloads[0].State != STATE_DOWNLOADING {
		t.Errorf("#1 was not downloading - %s instead ", q.Downloads[0].State)
		t.Log(q.String())
	}
	if q.Downloads[1].State != STATE_QUEUED {
		t.Errorf("#2 is not queued - %s instead", q.Downloads[1].State)
		t.Log(q.String())
	}
	if q.Downloads[2].State != STATE_QUEUED {
		t.Errorf("#3 is not queued - %s instead", q.Downloads[2].State)
		t.Log(q.String())
	}
	if q.Downloads[3].State != STATE_DOWNLOADING {
		t.Errorf("#4 is not downloading - %s instead", q.Downloads[3].State)
		t.Log(q.String())
	}

	// this should start no more, as one is still going
	q.startQueued(1)
	time.Sleep(time.Millisecond * 100)
	if q.Downloads[0].State != STATE_DOWNLOADING {
		t.Errorf("#1 was not downloading - %s instead ", q.Downloads[0].State)
		t.Log(q.String())
	}
	if q.Downloads[1].State != STATE_QUEUED {
		t.Errorf("#2 is not queued - %s instead", q.Downloads[1].State)
		t.Log(q.String())
	}
	if q.Downloads[2].State != STATE_QUEUED {
		t.Errorf("#3 is not queued - %s instead", q.Downloads[2].State)
		t.Log(q.String())
	}
	if q.Downloads[3].State != STATE_DOWNLOADING {
		t.Errorf("#4 is not downloading - %s instead", q.Downloads[3].State)
		t.Log(q.String())
	}

	// wait until the two finish, check
	time.Sleep(time.Second * 5.0)
	if q.Downloads[0].State != STATE_COMPLETE {
		t.Errorf("#1 was not complete - %s instead ", q.Downloads[0].State)
		t.Log(q.String())
	}
	if q.Downloads[1].State != STATE_QUEUED {
		t.Errorf("#2 is not queued - %s instead", q.Downloads[1].State)
		t.Log(q.String())
	}
	if q.Downloads[2].State != STATE_QUEUED {
		t.Errorf("#3 is not queued - %s instead", q.Downloads[2].State)
		t.Log(q.String())
	}
	if q.Downloads[3].State != STATE_COMPLETE {
		t.Errorf("#4 is not complete - %s instead", q.Downloads[3].State)
		t.Log(q.String())
	}

	// this should start one more, as one is still going
	q.startQueued(1)
	time.Sleep(time.Millisecond * 100)
	if q.Downloads[0].State != STATE_COMPLETE {
		t.Errorf("#1 was not complete - %s instead ", q.Downloads[0].State)
		t.Log(q.String())
	}
	if q.Downloads[1].State != STATE_DOWNLOADING {
		t.Errorf("#2 is not downloading - %s instead", q.Downloads[1].State)
		t.Log(q.String())
	}
	if q.Downloads[2].State != STATE_QUEUED {
		t.Errorf("#3 is not queued - %s instead", q.Downloads[2].State)
		t.Log(q.String())
	}
	if q.Downloads[3].State != STATE_COMPLETE {
		t.Errorf("#4 is not complete - %s instead", q.Downloads[3].State)
		t.Log(q.String())
	}

	// this should start no more, as one is still going
	q.startQueued(1)
	time.Sleep(time.Millisecond * 100)
	if q.Downloads[0].State != STATE_COMPLETE {
		t.Errorf("#1 was not complete - %s instead ", q.Downloads[0].State)
		t.Log(q.String())
	}
	if q.Downloads[1].State != STATE_DOWNLOADING {
		t.Errorf("#2 is not downloading - %s instead", q.Downloads[1].State)
		t.Log(q.String())
	}
	if q.Downloads[2].State != STATE_QUEUED {
		t.Errorf("#3 is not queued - %s instead", q.Downloads[2].State)
		t.Log(q.String())
	}
	if q.Downloads[3].State != STATE_COMPLETE {
		t.Errorf("#4 is not complete - %s instead", q.Downloads[3].State)
		t.Log(q.String())
	}

	// but if we allow two per domain, the other queued one will start
	q.startQueued(2)
	time.Sleep(time.Millisecond * 100)
	if q.Downloads[0].State != STATE_COMPLETE {
		t.Errorf("#1 was not complete - %s instead ", q.Downloads[0].State)
		t.Log(q.String())
	}
	if q.Downloads[1].State != STATE_DOWNLOADING {
		t.Errorf("#2 is not downloading - %s instead", q.Downloads[1].State)
		t.Log(q.String())
	}
	if q.Downloads[2].State != STATE_DOWNLOADING {
		t.Errorf("#3 is not downloading - %s instead", q.Downloads[2].State)
		t.Log(q.String())
	}
	if q.Downloads[3].State != STATE_COMPLETE {
		t.Errorf("#4 is not complete - %s instead", q.Downloads[3].State)
		t.Log(q.String())
	}

}

func TestUpdateMetadataPlaylist(t *testing.T) {

	output := `
start of log...
[download] Downloading playlist: nice_user
[RedGifsUser] nice_user: Downloading JSON metadata page 1
[RedGifsUser] nice_user: Downloading JSON metadata page 2
[RedGifsUser] nice_user: Downloading JSON metadata page 3
[RedGifsUser] nice_user: Downloading JSON metadata page 4
[RedGifsUser] nice_user: Downloading JSON metadata page 5
[RedGifsUser] nice_user: Downloading JSON metadata page 6
[info] Writing playlist metadata as JSON to: nice_user [nice_user].info.json
[RedGifsUser] playlist nice_user: Downloading 3 videos
[download] Downloading video 1 of 3
[info] wrongpreciouschrysomelid: Downloading 1 format(s): hd
[info] Writing video metadata as JSON to: Splendid Wonderful Speaker Power Chocolate Drop [wrongpreciouschrysomelid].info.json
[download] Destination: Splendid Wonderful Speaker Power Chocolate Drop [wrongpreciouschrysomelid].mp4
[download]   0.0% of 4.96MiB at Unknown speed ETA Unknown
[download]   0.1% of 4.96MiB at  1.76MiB/s ETA 00:02
[download]  20.1% of 4.96MiB at  7.28MiB/s ETA 00:00
[download]  40.3% of 4.96MiB at 10.06MiB/s ETA 00:00
[download]  80.6% of 4.96MiB at 14.93MiB/s ETA 00:00
[download] 100% of 4.96MiB at 17.33MiB/s ETA 00:00
[download] 100% of 4.96MiB in 00:00
[download] Downloading video 2 of 3
[info] silentnaughtyborzoi: Downloading 1 format(s): hd
[info] Writing video metadata as JSON to: Splendid Printer Tray Computer Outdoor Window Wonderful [silentnaughtyborzoi].info.json
[download] Destination: Splendid Printer Tray Computer Outdoor Window Wonderful [silentnaughtyborzoi].mp4
[download]   0.0% of 5.81MiB at 896.03KiB/s ETA 00:06
[download]   0.1% of 5.81MiB at  1.28MiB/s ETA 00:04
[download]   0.1% of 5.81MiB at  1.59MiB/s ETA 00:03
[download]  34.4% of 5.81MiB at  9.90MiB/s ETA 00:00
[download]  68.8% of 5.81MiB at 12.49MiB/s ETA 00:00
[download] 100% of 5.81MiB at 15.77MiB/s ETA 00:00
[download] 100% of 5.81MiB in 00:00
[download] Downloading video 3 of 3
[info] mammothremarkablewhooper: Downloading 1 format(s): hd
[info] Writing video metadata as JSON to: Porthole Splendid Close Up Gun Gunshot Window Wonderful [mammothremarkablewhooper].info.json
[download] Destination: Porthole Splendid Close Up Gun Gunshot Window Wonderful [mammothremarkablewhooper].mp4
[download]   0.0% of 2.89MiB at Unknown speed ETA Unknown
[download]   0.1% of 2.89MiB at  1.77MiB/s ETA 00:01
[download]   0.2% of 2.89MiB at  2.26MiB/s ETA 00:01
[download]  34.5% of 2.89MiB at  8.23MiB/s ETA 00:00
[download]  69.1% of 2.89MiB at 11.63MiB/s ETA 00:00
[download] 100% of 2.89MiB at 14.25MiB/s ETA 00:00
[download] 100% of 2.89MiB in 00:00
[info] Writing updated playlist metadata as JSON to: nice_user [nice_user].info.json
[download] Finished downloading playlist: nice_user
`
	newD := Download{}

	lines := strings.Split(output, "\n")
	for _, l := range lines {
		// t.Log(l)
		newD.updateMetadata(l)
	}

	if len(newD.Files) != 3 {
		t.Errorf("%d files, not 3", len(newD.Files))
	} else {
		if newD.Files[0] != "Splendid Wonderful Speaker Power Chocolate Drop [wrongpreciouschrysomelid].mp4" {
			t.Error("Wrong 1st file")
		}
		if newD.Files[1] != "Splendid Printer Tray Computer Outdoor Window Wonderful [silentnaughtyborzoi].mp4" {
			t.Error("Wrong 2nd file")
		}
		if newD.Files[2] != "Porthole Splendid Close Up Gun Gunshot Window Wonderful [mammothremarkablewhooper].mp4" {
			t.Error("Wrong 3rd file")
		}
	}

	if newD.PlaylistTotal != 3 {
		t.Errorf("playlist has total %d should be 3", newD.PlaylistTotal)
	}

}

func TestUpdateMetadataSingle(t *testing.T) {

	output := `
[youtube] 2WoDQBhJCVQ: Downloading webpage
[youtube] 2WoDQBhJCVQ: Downloading android player API JSON
[info] 2WoDQBhJCVQ: Downloading 1 format(s): 137+140
[info] Writing video metadata as JSON to: The Greatest Shot In Television [2WoDQBhJCVQ].info.json
[download] Destination: The Greatest Shot In Television [2WoDQBhJCVQ].f137.mp4
[download]   0.0% of 12.82MiB at 510.94KiB/s ETA 00:26
[download]   0.0% of 12.82MiB at 966.50KiB/s ETA 00:13
[download]   0.1% of 12.82MiB at  1.54MiB/s ETA 00:08
[download]   0.1% of 12.82MiB at  2.75MiB/s ETA 00:04
[download]   0.2% of 12.82MiB at  1.30MiB/s ETA 00:09
[download]  77.5% of 12.82MiB at  2.54MiB/s ETA 00:01
[download]  79.4% of 12.82MiB at  3.89MiB/s ETA 00:00
[download]  83.3% of 12.82MiB at  6.44MiB/s ETA 00:00
[download]  91.1% of 12.82MiB at 10.28MiB/s ETA 00:00
[download] 100% of 12.82MiB at 12.77MiB/s ETA 00:00
[download] 100% of 12.82MiB in 00:01
[download] Destination: The Greatest Shot In Television [2WoDQBhJCVQ].f140.m4a
[download]   0.1% of 1.10MiB at 286.46KiB/s ETA 00:03
[download]   0.3% of 1.10MiB at 716.49KiB/s ETA 00:01
[download]   0.6% of 1.10MiB at  1.42MiB/s ETA 00:00
[download]  91.0% of 1.10MiB at  6.67MiB/s ETA 00:00
[download] 100% of 1.10MiB at  7.06MiB/s ETA 00:00
[download] 100% of 1.10MiB in 00:00
[Merger] Merging formats into "The Greatest Shot In Television [2WoDQBhJCVQ].mp4"
Deleting original file The Greatest Shot In Television [2WoDQBhJCVQ].f137.mp4 (pass -k to keep)
Deleting original file The Greatest Shot In Television [2WoDQBhJCVQ].f140.m4a (pass -k to keep)
`
	newD := Download{}

	lines := strings.Split(output, "\n")
	for _, l := range lines {
		// t.Log(l)
		newD.updateMetadata(l)
	}

	if len(newD.Files) != 1 {
		t.Errorf("%d files, not 1", len(newD.Files))
	} else {
		if newD.Files[0] != "The Greatest Shot In Television [2WoDQBhJCVQ].mp4" {
			t.Error("Wrong 1st file")
		}
	}
	if newD.PlaylistTotal != 0 {
		t.Error("playlist detected but should not be")
	}

}
