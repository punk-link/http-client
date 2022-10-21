package httpclient

func alignSlice[T any](slice []T, divisor int) ([]T, []T) {
	var unalignedItems []T
	alignedItems := make([]T, 0)

	if len(slice) < divisor {
		unalignedItems = slice
		return alignedItems, unalignedItems
	}

	extraElements := len(slice) % divisor

	alignedItems = slice[0 : len(slice)-extraElements]
	unalignedItems = slice[len(slice)-extraElements:]

	return alignedItems, unalignedItems
}
