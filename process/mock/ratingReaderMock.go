package mock

type RatingReaderMock struct {
	GetRatingCalled                  func(string) uint32
	UpdateRatingFromTempRatingCalled func(string)
	RatingsMap                       map[string]uint32
}

func (rrm *RatingReaderMock) GetRating(pk string) uint32 {
	if rrm.GetRatingCalled != nil {
		return rrm.GetRatingCalled(pk)
	}

	return 0
}

func (rrm *RatingReaderMock) UpdateRatingFromTempRating(pk string) {
	if rrm.UpdateRatingFromTempRatingCalled != nil {
		rrm.UpdateRatingFromTempRatingCalled(pk)
	}
}

func (rrm *RatingReaderMock) IsInterfaceNil() bool {
	return rrm == nil
}
