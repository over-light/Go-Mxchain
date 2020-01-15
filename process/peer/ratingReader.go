package peer

type RatingReader struct {
	getRating                  func(string) uint32
	updateRatingFromTempRating func([]string)
}

//GetRating returns the Rating for the specified public key
func (bsr *RatingReader) GetRating(pk string) uint32 {
	rating := bsr.getRating(pk)
	return rating
}

//UpdateRatingFromTempRating returns the TempRating for the specified public key
func (bsr *RatingReader) UpdateRatingFromTempRating(pks []string) {
	bsr.updateRatingFromTempRating(pks)
}

//IsInterfaceNil checks if the underlying object is nil
func (bsr *RatingReader) IsInterfaceNil() bool {
	return bsr == nil
}
