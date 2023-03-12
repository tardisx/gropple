//go:build testdata

package download

import "github.com/tardisx/gropple/config"

func (m *Manager) AddStressTestData(c *config.ConfigService) {

	urls := []string{
		"https://www.youtube.com/watch?v=qG_rRkuGBW8",
		"https://www.youtube.com/watch?v=ZUzhZpQAU40",
		"https://www.youtube.com/watch?v=kVxM3eRWGak",
		"https://www.youtube.com/watch?v=pl-y9869y0w",
		"https://vimeo.com/783453809",
		"https://www.youtube.com/watch?v=Uw4NEPE4l3A",
		"https://www.youtube.com/watch?v=2RF0lcTuuYE",
		"https://www.youtube.com/watch?v=lymwNQY0dus",
		"https://www.youtube.com/watch?v=NTc-I4Z_duc",
		"https://www.youtube.com/watch?v=wNSm1TJ84Ac",
		"https://vimeo.com/786570322",
	}
	for _, u := range urls {
		d := NewDownload(u, c.Config)
		d.DownloadProfile = *c.Config.ProfileCalled("standard video")
		m.AddDownload(d)
		m.Queue(d)
	}
}
