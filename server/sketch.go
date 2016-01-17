package server

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	pb "github.com/seiflotfy/skizze/datamodel"

	"golang.org/x/net/context"
)

func (s *serverStruct) CreateSketch(ctx context.Context, in *pb.Sketch) (*pb.Sketch, error) {
	info := pb.NewEmptyInfo()
	info.Name = in.GetName()
	info.Type = strings.ToLower(in.GetType().String())
	info.Properties.Rank = uint(in.Defaults.GetRank())
	info.Properties.Capacity = uint(in.Defaults.GetCapacity())
	if err := s.manager.CreateSketch(info); err != nil {
		return nil, err
	}
	return in, nil
}

func (s *serverStruct) Add(ctx context.Context, in *pb.AddRequest) (*pb.AddReply, error) {
	info := pb.NewEmptyInfo()
	if dom := in.GetDomain(); dom != nil {
		fmt.Println(dom)
		info.Name = dom.GetName()
		info.Type = pb.DOM
		err := s.manager.AddToDomain(info.Name, in.GetValues())
		if err != nil {
			return nil, err
		}
	} else if sketch := in.GetSketch(); sketch != nil {
		info.Name = sketch.GetName()
		info.Type = strings.ToLower(sketch.GetType().String())
		err := s.manager.AddToSketch(info.ID(), in.GetValues())
		if err != nil {
			return nil, err
		}
	}
	return &pb.AddReply{}, nil
}

func (s *serverStruct) GetMembership(ctx context.Context, in *pb.GetRequest) (*pb.GetMembershipReply, error) {
	sketch := in.GetSketch()
	info := pb.NewEmptyInfo()
	info.Name = sketch.GetName()
	info.Type = strings.ToLower(pb.SketchType_MEMB.String())
	res, err := s.manager.GetFromSketch(info.ID(), in.GetValues())
	if err != nil {
		return nil, err
	}
	reply := &pb.GetMembershipReply{}
	values := res.([]*pb.Member)
	// FIXME: return in same order
	for _, v := range values {
		reply.Memberships = append(reply.Memberships, &pb.Membership{
			Value:    proto.String(v.Key),
			IsMember: proto.Bool(v.Member),
		})
	}
	return reply, nil
}

func (s *serverStruct) GetFrequency(ctx context.Context, in *pb.GetRequest) (*pb.GetFrequencyReply, error) {
	sketch := in.GetSketch()
	info := pb.NewEmptyInfo()
	info.Name = sketch.GetName()
	info.Type = strings.ToLower(pb.SketchType_FREQ.String())
	res, err := s.manager.GetFromSketch(info.ID(), in.GetValues())
	if err != nil {
		return nil, err
	}
	reply := &pb.GetFrequencyReply{}
	// FIXME: return in same order
	for k, v := range res.(map[string]uint) {
		reply.Frequencies = append(reply.Frequencies, &pb.Frequency{
			Value: proto.String(k),
			Count: proto.Int64(int64(v)),
		})
	}
	return reply, nil
}

func (s *serverStruct) GetCardinality(ctx context.Context, in *pb.GetRequest) (*pb.GetCardinalityReply, error) {
	sketch := in.GetSketch()
	info := pb.NewEmptyInfo()
	info.Name = sketch.GetName()
	info.Type = strings.ToLower(pb.SketchType_CARD.String())
	res, err := s.manager.GetFromSketch(info.ID(), in.GetValues())
	if err != nil {
		return nil, err
	}
	reply := &pb.GetCardinalityReply{
		Cardinality: proto.Int64(int64(res.(uint))),
	}
	return reply, nil
}

func (s *serverStruct) GetRank(ctx context.Context, in *pb.GetRequest) (*pb.GetRankReply, error) {
	sketch := in.GetSketch()
	info := pb.NewEmptyInfo()
	info.Name = sketch.GetName()
	info.Type = strings.ToLower(pb.SketchType_RANK.String())
	res, err := s.manager.GetFromSketch(info.ID(), in.GetValues())
	if err != nil {
		return nil, err
	}
	reply := &pb.GetRankReply{}
	for _, v := range res.([]*pb.Element) {
		reply.Ranks = append(reply.Ranks, &pb.Rank{
			Value: proto.String(v.Key),
			Count: proto.Int64(int64(v.Count)),
		})
	}
	return reply, nil
}

func (s *serverStruct) DeleteSketch(ctx context.Context, in *pb.Sketch) (*pb.Empty, error) {
	info := pb.NewEmptyInfo()
	info.Name = in.GetName()
	info.Type = strings.ToLower(in.GetType().String())
	return &pb.Empty{}, s.manager.DeleteSketch(info.ID())
}

func (s *serverStruct) ListAll(ctx context.Context, in *pb.Empty) (*pb.ListReply, error) {
	sketches := s.manager.GetSketches()
	filtered := &pb.ListReply{}
	for _, v := range sketches {
		typ := pb.SketchType_UNKNOWN
		switch v[1] {
		case pb.CML:
			typ = pb.SketchType_FREQ
		case pb.TopK:
			typ = pb.SketchType_RANK
		case pb.HLLPP:
			typ = pb.SketchType_CARD
		case pb.Bloom:
			typ = pb.SketchType_MEMB
		default:
			continue
		}
		filtered.Sketches = append(filtered.Sketches, &pb.Sketch{Name: proto.String(v[0]), Type: &typ})
	}
	return filtered, nil
}
