//go:build testdata

package download

import "github.com/tardisx/gropple/config"

func (m *Manager) AddStressTestData(c *config.ConfigService) {

	urls := []string{
		"https://www.youtube.com/watch?v=qG_rRkuGBW8",
		"https://www.youtube.com/watch?v=ZUzhZpQAU40",
		"https://www.youtube.com/watch?v=kVxM3eRWGak",
		"https://www.youtube.com/watch?v=pl-y9869y0w",
		"https://www.youtube.com/watch?v=Uw4NEPE4l3A",
		"https://www.youtube.com/watch?v=2RF0lcTuuYE",
		"https://www.youtube.com/watch?v=lymwNQY0dus",
		"https://www.youtube.com/watch?v=NTc-I4Z_duc",
		"https://www.youtube.com/watch?v=wNSm1TJ84Ac",
		"https://www.youtube.com/watch?v=tyixMpuGEL8",
		"https://www.youtube.com/watch?v=VnxbkH_3E_4",
		"https://www.youtube.com/watch?v=VStscvYLYLs",
		"https://www.youtube.com/watch?v=vYMiSz-WlEY",

		"https://vimeo.com/786570322",
		"https://vimeo.com/783453809",

		"https://www.gamespot.com/videos/survival-fps-how-metro-2033-solidified-a-subgenre/2300-6408243/",
		"https://www.gamespot.com/videos/dirt-3-right-back-where-you-started-gameplay-movie/2300-6314712/",
		"https://www.gamespot.com/videos/the-b-list-driver-san-francisco/2300-6405593/",

		"https://www.imdb.com/video/vi1914750745/?listId=ls053181649&ref_=hm_hp_i_hero-video-1_1",
		"https://www.imdb.com/video/vi3879585561/?listId=ls053181649&ref_=vp_pl_ap_6",
		"https://www.imdb.com/video/vi54445849/?listId=ls053181649&ref_=vp_nxt_btn",
	}
	for _, u := range urls {
		d := NewDownload(u, c.Config)
		d.DownloadProfile = *c.Config.ProfileCalled("standard video")
		m.AddDownload(d)
		m.Queue(d)
	}
}
