package template

import "testing"

func TestResolvedQuad_FourPoints(t *testing.T) {
	b := TextBox{Quad: []Point{
		{0.1, 0.2}, {0.9, 0.2}, {0.9, 0.8}, {0.1, 0.8},
	}}
	q, err := b.ResolvedQuad()
	if err != nil {
		t.Fatal(err)
	}
	if q[2].X != 0.9 || q[2].Y != 0.8 {
		t.Fatalf("br = %v", q[2])
	}
}

func TestResolvedQuad_ThreePointWrapper(t *testing.T) {
	b := TextBox{Parallelogram: []Point{
		{0.1, 0.2}, {0.5, 0.2}, {0.1, 0.6},
	}}
	q, err := b.ResolvedQuad()
	if err != nil {
		t.Fatal(err)
	}
	// br = tl + (tr-tl) + (bl-tl) = (0.5, 0.6)
	if q[2].X != 0.5 || q[2].Y != 0.6 {
		t.Fatalf("implied br = %+v, want {0.5 0.6}", q[2])
	}
	if q[3].X != 0.1 || q[3].Y != 0.6 {
		t.Fatalf("bl = %+v", q[3])
	}
}
