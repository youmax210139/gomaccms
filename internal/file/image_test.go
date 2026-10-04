package file

import "testing"

func TestIsImage(t *testing.T) {
	for _, name := range []string{"a.jpg", "a.JPEG", "b.png", "c.webp", "d.gif", "dir/e.GIF"} {
		if !IsImage(name) {
			t.Errorf("%s must be accepted", name)
		}
	}
	for _, name := range []string{"a.html", "b.svg", "c.php", "d", "e.gif.html", "f.js"} {
		if IsImage(name) {
			t.Errorf("%s must be rejected", name)
		}
	}
}
