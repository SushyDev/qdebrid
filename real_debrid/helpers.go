package real_debrid

func GetIdsByHash(hash string) ([]string, error) {
	instantAvailability, err := InstantAvailability(hash)
	if err != nil {
		return nil, err
	}

	var ids []string
	for key := range instantAvailability {
		ids = append(ids, key)
	}

	return ids, nil
}
