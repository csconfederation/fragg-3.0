package eco

import "github.com/csconfederation/fragg-3.0/model"

// Stats is the round's rows by Steam64.
func (r *Round) Stats() map[uint64]*model.RoundStats { return r.stats }
