package typeutils

import "go/types"

func GetObjForPtrToNamedType(typ types.Type) types.Object {
	ptr, ok := typ.(*types.Pointer)
	if !ok {
		return nil
	}

	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return nil
	}

	return named.Obj()
}
