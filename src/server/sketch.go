package server

import (
	"datamodel"
	pb "datamodel/protobuf"
	"utils"

	"github.com/gogo/protobuf/proto"
	"golang.org/x/net/context"
)

func (s *serverStruct) CreateSketch(ctx context.Context, in *pb.Sketch) (*pb.Sketch, error) {
	info := &datamodel.Info{Sketch: in}
	if err := s.manager.CreateSketch(info); err != nil {
		return nil, err
	}
	return in, nil
}

func (s *serverStruct) Add(ctx context.Context, in *pb.AddRequest) (*pb.AddReply, error) {
	info := datamodel.NewEmptyInfo()
	// FIXME: use domain or sketch directly and stop casting to Info
	if dom := in.GetDomain(); dom != nil {
		info.Name = dom.Name
		err := s.manager.AddToDomain(info.GetName(), in.GetValues())
		if err != nil {
			return nil, err
		}
	} else if sketch := in.GetSketch(); sketch != nil {
		info := &datamodel.Info{Sketch: sketch}
		err := s.manager.AddToSketch(info.ID(), in.GetValues())
		if err != nil {
			return nil, err
		}
	}
	return &pb.AddReply{}, nil
}

func (s *serverStruct) GetMembership(ctx context.Context, in *pb.GetRequest) (*pb.GetMembershipReply, error) {
	reply := &pb.GetMembershipReply{}

	for _, sketch := range in.GetSketches() {
		info := &datamodel.Info{Sketch: sketch}
		res, err := s.manager.GetFromSketch(info.ID(), in.GetValues())
		if err != nil {
			return nil, err
		}
		result := &pb.MembershipResult{}
		values := res.([]*datamodel.Member)
		// FIXME: return in same order
		for _, v := range values {
			result.Memberships = append(result.Memberships, &pb.Membership{
				Value:    utils.Stringp(v.Key),
				IsMember: utils.Boolp(v.Member),
			})
		}
		reply.Results = append(reply.Results, result)
	}
	return reply, nil
}

func (s *serverStruct) GetFrequency(ctx context.Context, in *pb.GetRequest) (*pb.GetFrequencyReply, error) {
	reply := &pb.GetFrequencyReply{}

	for _, sketch := range in.GetSketches() {
		info := &datamodel.Info{Sketch: sketch}
		res, err := s.manager.GetFromSketch(info.ID(), in.GetValues())
		if err != nil {
			return nil, err
		}
		result := &pb.FrequencyResult{}
		// FIXME: return in same order
		for k, v := range res.(map[string]uint) {
			result.Frequencies = append(result.Frequencies, &pb.Frequency{
				Value: proto.String(k),
				Count: proto.Int64(int64(v)),
			})
		}
		reply.Results = append(reply.Results, result)
	}
	return reply, nil
}

func (s *serverStruct) GetCardinality(ctx context.Context, in *pb.GetRequest) (*pb.GetCardinalityReply, error) {
	reply := &pb.GetCardinalityReply{}

	for _, sketch := range in.GetSketches() {
		info := &datamodel.Info{Sketch: sketch}
		res, err := s.manager.GetFromSketch(info.ID(), in.GetValues())
		if err != nil {
			return nil, err
		}
		result := &pb.CardinalityResult{
			Cardinality: proto.Int64(int64(res.(uint))),
		}
		reply.Results = append(reply.Results, result)
	}
	return reply, nil
}

func (s *serverStruct) GetRankings(ctx context.Context, in *pb.GetRequest) (*pb.GetRankingsReply, error) {
	reply := &pb.GetRankingsReply{}

	for _, sketch := range in.GetSketches() {
		info := &datamodel.Info{Sketch: sketch}

		res, err := s.manager.GetFromSketch(info.ID(), in.GetValues())
		if err != nil {
			return nil, err
		}
		result := &pb.RankingsResult{}
		for _, v := range res.([]*datamodel.Element) {
			result.Rankings = append(result.Rankings, &pb.Rank{
				Value: proto.String(v.Key),
				Count: proto.Int64(int64(v.Count)),
			})
		}
		reply.Results = append(reply.Results, result)
	}
	return reply, nil
}

func (s *serverStruct) DeleteSketch(ctx context.Context, in *pb.Sketch) (*pb.Empty, error) {
	info := &datamodel.Info{Sketch: in}
	return &pb.Empty{}, s.manager.DeleteSketch(info.ID())
}

func (s *serverStruct) ListAll(ctx context.Context, in *pb.Empty) (*pb.ListReply, error) {
	sketches := s.manager.GetSketches()
	filtered := &pb.ListReply{}
	for _, v := range sketches {
		var typ pb.SketchType
		switch v[1] {
		case datamodel.CML:
			typ = pb.SketchType_FREQ
		case datamodel.TopK:
			typ = pb.SketchType_RANK
		case datamodel.HLLPP:
			typ = pb.SketchType_CARD
		case datamodel.Bloom:
			typ = pb.SketchType_MEMB
		default:
			continue
		}
		filtered.Sketches = append(filtered.Sketches, &pb.Sketch{Name: proto.String(v[0]), Type: &typ})
	}
	return filtered, nil
}

func (s *serverStruct) GetSketch(ctx context.Context, in *pb.Sketch) (*pb.Sketch, error) {
	var err error
	info := &datamodel.Info{Sketch: in}
	if info, err = s.manager.GetSketch(info.ID()); err != nil {
		return in, err
	}
	return in, nil
}

func (s *serverStruct) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	sketches := s.manager.GetSketches()
	filtered := &pb.ListReply{}
	for _, v := range sketches {
		var typ pb.SketchType
		switch v[1] {
		case datamodel.CML:
			typ = pb.SketchType_FREQ
		case datamodel.TopK:
			typ = pb.SketchType_RANK
		case datamodel.HLLPP:
			typ = pb.SketchType_CARD
		case datamodel.Bloom:
			typ = pb.SketchType_MEMB
		default:
			continue
		}
		if in.GetType() == typ {
			filtered.Sketches = append(filtered.Sketches, &pb.Sketch{Name: proto.String(v[0]), Type: &typ})
		}
	}
	return filtered, nil
}