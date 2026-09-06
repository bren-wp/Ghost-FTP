package remote

func (s *uploadSourceSnapshot) Size() int64 {
	if s == nil || s.initial == nil {
		return 0
	}
	size := s.initial.Size()
	if size < 0 {
		return 0
	}
	return size
}
