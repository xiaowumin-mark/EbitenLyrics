//go:build !windows

package font

func (m *FontManager) searchFamilyPlatform(family string) ([]fontRecord, bool, error) {
	return nil, false, nil
}
