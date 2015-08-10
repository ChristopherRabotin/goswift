package goswift

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func newPerishableInfo(hits int) *PerishableInfo {
	expiry := time.Now().Add(time.Second * 30)
	return &PerishableInfo{Hits: hits, Expires: expiry}
}

// TestPerishable tests all of features of the redis interface.
func TestPerishable(t *testing.T) {
	Convey("The Perishable auth tests, ", t, func() {
		Convey("Given a PerishableInfo", func() {
			Convey("All nominal", func() {
				p := newPerishableInfo(0)
				So(p.isValid(), ShouldEqual, true)
			})
			Convey("If the hits is one less than the  NonceLimit, it should be valid", func() {
				p := newPerishableInfo(NonceLimit - 1)
				So(p.isValid(), ShouldEqual, true)
			})
			Convey("If the hits is equal to NonceLimit, it should not be valid", func() {
				p := newPerishableInfo(NonceLimit)
				So(p.isValid(), ShouldEqual, false)
			})
			Convey("If the hits is one above the  NonceLimit, it should be valid", func() {
				p := newPerishableInfo(NonceLimit + 1)
				So(p.isValid(), ShouldEqual, false)
			})
			Convey("If the expiry time is right after now, it should be valid.", func() {
				p := newPerishableInfo(0)
				p.Expires = time.Now().Add(time.Second * 1)
				So(p.isValid(), ShouldEqual, true)
			})
			Convey("If the expiry time is now, it should not be valid.", func() {
				p := newPerishableInfo(0)
				p.Expires = time.Now()
				So(p.isValid(), ShouldEqual, false)
			})
			Convey("If the expiry time is right before now, it should not be valid.", func() {
				p := newPerishableInfo(0)
				p.Expires = time.Now().Add(time.Millisecond * -1)
				So(p.isValid(), ShouldEqual, false)
			})

		})

	})
}
