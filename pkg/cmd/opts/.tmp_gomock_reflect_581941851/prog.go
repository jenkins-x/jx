
package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"path"
	"reflect"

	"github.com/petergtz/pegomock/model"
	"github.com/petergtz/pegomock/modelgen/gomock"

	pkg_ "github.com/jenkins-x/jx/pkg/cmd/opts"
)

func main() {
	its := []struct{
		sym string
		typ reflect.Type
	}{
		
		{ "CommonOptions", reflect.TypeOf((*pkg_.CommonOptions)(nil)).Elem()},
		
	}
	pkg := &model.Package{
		// NOTE: This behaves contrary to documented behaviour if the
		// package name is not the final component of the import path.
		// The reflect package doesn't expose the package name, though.
		Name: path.Base("github.com/jenkins-x/jx/pkg/cmd/opts"),
	}

	for _, it := range its {
		intf, err := gomock.InterfaceFromInterfaceType(it.typ)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Reflection: %v\n", err)
			os.Exit(1)
		}
		intf.Name = it.sym
		pkg.Interfaces = append(pkg.Interfaces, intf)
	}
	if err := gob.NewEncoder(os.Stdout).Encode(pkg); err != nil {
		fmt.Fprintf(os.Stderr, "gob encode: %v\n", err)
		os.Exit(1)
	}
}
