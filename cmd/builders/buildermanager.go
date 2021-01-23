package builder

import (
	"log"
)

type Builder struct {
	BuilderName string
}

type BuilderManager struct {
	Builders []*Builder
}

func (m *BuilderManager) PrintBuilders() {
	for _, builder := range m.Builders {
		log.Println(builder.BuilderName)
	}
}

var Manager = &BuilderManager{
	Builders: []*Builder{
		NodeBuilder,
	},
}
