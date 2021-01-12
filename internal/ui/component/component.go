package component

import "github.com/gotk3/gotk3/gtk"

type Component struct {
	builder *gtk.Builder
}

func builder(res string) *gtk.Builder {
	ibuilder, err := gtk.BuilderNewFromResource(res)
	if err != nil {
		panic(err)
	}
	return ibuilder
}

func (c *Component) GladeWidget(name string) gtk.IWidget {
	obj, err := c.builder.GetObject(name)
	if err != nil {
		panic(err)
	}

	w, ok := obj.(gtk.IWidget)
	if !ok {
		panic("error casting obj")
	}

	return w
}

func New(res string) *Component {
	return &Component{builder: builder(res)}
}
