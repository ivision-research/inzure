package inzure

type FromAzurer[T any] interface {
	FromAzure(T)
}

func fromAzureSetter[Az any, T FromAzurer[*Az]](into T, az *Az) {
	into.FromAzure(az)
}

func gSliceFromPtrSetterPtrs[Az any, T any](into *[]T, from *[]*Az, set func(*T, *Az)) {
	if from == nil || len(*from) == 0 {
		*into = make([]T, 0)
		return
	}
	src := *from
	newSlice := make([]T, len(src))
	for i, azelem := range src {
		set(&newSlice[i], azelem)
	}
	*into = newSlice
}

func gSliceFromPtrSetter[Az any, T any](into *[]T, from *[]Az, set func(*T, *Az)) {
	if from == nil || len(*from) == 0 {
		*into = make([]T, 0)
		return
	}
	src := *from
	newSlice := make([]T, len(src))
	for i, azelem := range src {
		set(&newSlice[i], &azelem)
	}
	*into = newSlice
}

func gValFromPtrDefault[T any](into *T, from *T, def T) {
	if from == nil {
		*into = def
		return
	}
	*into = *from
}

func gValFromPtr[T any](into *T, from *T) {
	if from == nil {
		return
	}
	*into = *from
}

func gValFromPtrFromAzure[Az any, T FromAzurer[*Az]](into T, from *Az) {
	if from == nil {
		return
	}
	into.FromAzure(from)
}

func gSliceFromPtr[T any](into *[]T, from *[]T) {
	if from == nil {
		return
	}
	newSlice := make([]T, len(*from))
	copy(newSlice, *from)
	*into = newSlice
}
