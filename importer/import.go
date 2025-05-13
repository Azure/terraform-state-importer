package importer

import (
	"fmt"
)

type Importer struct{}

func (importer *Importer) Import() {
	fmt.Println("import called with", importer)
}
