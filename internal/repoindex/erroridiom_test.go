package repoindex

import "testing"

func TestClassifyErrorIdiomLine(t *testing.T) {
	cases := map[string]string{
		`return fmt.Errorf("open db: %w", err)`: IdiomWrap,
		`	return err`:                           IdiomBare,
		`	return nil, err`:                      IdiomBare,
		`panic(err)`:                            IdiomPanic,
		`x := compute()`:                        "",
		`return errors.New("no idiom match")`:   "",
	}
	for line, want := range cases {
		if got := ClassifyErrorIdiomLine(line); got != want {
			t.Errorf("%q -> %q, want %q", line, got, want)
		}
	}
}

func TestDominantErrorIdiom(t *testing.T) {
	idx := &Index{}
	src := []byte(`package p
func a() error { return fmt.Errorf("a: %w", err) }
func b() error { return fmt.Errorf("b: %w", err) }
func c() error { return fmt.Errorf("c: %w", err) }
func d() error { return fmt.Errorf("d: %w", err) }
func e() error {
	return err
}
`)
	idx.countErrorIdioms(src)
	dom, share := idx.Conventions.DominantErrorIdiom()
	if dom != IdiomWrap || share < 0.79 {
		t.Fatalf("dominant = %s (%.2f), want wrap >= 0.8", dom, share)
	}
}
