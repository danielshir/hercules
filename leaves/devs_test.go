package leaves

import (
	"bytes"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/hercules.v7/internal/core"
	"gopkg.in/src-d/hercules.v7/internal/pb"
	"gopkg.in/src-d/hercules.v7/internal/plumbing"
	"gopkg.in/src-d/hercules.v7/internal/plumbing/identity"
	"gopkg.in/src-d/hercules.v7/internal/test"
	"gopkg.in/src-d/hercules.v7/internal/test/fixtures"
)

func fixtureDevs() *DevsAnalysis {
	d := DevsAnalysis{}
	d.Initialize(test.Repository)
	people := [...]string{"one@srcd", "two@srcd"}
	d.reversedPeopleDict = people[:]
	return &d
}

func TestDevsMeta(t *testing.T) {
	d := fixtureDevs()
	assert.Equal(t, d.Name(), "Devs")
	assert.Equal(t, len(d.Provides()), 0)
	assert.Equal(t, len(d.Requires()), 6)
	assert.Equal(t, d.Requires()[0], identity.DependencyAuthor)
	assert.Equal(t, d.Requires()[1], plumbing.DependencyTreeChanges)
	assert.Equal(t, d.Requires()[2], plumbing.DependencyFileDiff)
	assert.Equal(t, d.Requires()[3], plumbing.DependencyBlobCache)
	assert.Equal(t, d.Requires()[4], plumbing.DependencyDay)
	assert.Equal(t, d.Requires()[5], plumbing.DependencyLanguages)
	assert.Equal(t, d.Flag(), "devs")
	assert.Len(t, d.ListConfigurationOptions(), 1)
	assert.Equal(t, d.ListConfigurationOptions()[0].Name, ConfigDevsConsiderEmptyCommits)
	assert.Equal(t, d.ListConfigurationOptions()[0].Flag, "empty-commits")
	assert.Equal(t, d.ListConfigurationOptions()[0].Type, core.BoolConfigurationOption)
	assert.Equal(t, d.ListConfigurationOptions()[0].Default, false)
	assert.True(t, len(d.Description()) > 0)
}

func TestDevsRegistration(t *testing.T) {
	summoned := core.Registry.Summon((&DevsAnalysis{}).Name())
	assert.Len(t, summoned, 1)
	assert.Equal(t, summoned[0].Name(), "Devs")
	leaves := core.Registry.GetLeaves()
	matched := false
	for _, tp := range leaves {
		if tp.Flag() == (&DevsAnalysis{}).Flag() {
			matched = true
			break
		}
	}
	assert.True(t, matched)
}

func TestDevsConfigure(t *testing.T) {
	devs := DevsAnalysis{}
	facts := map[string]interface{}{}
	facts[ConfigDevsConsiderEmptyCommits] = true
	devs.Configure(facts)
	assert.Equal(t, devs.ConsiderEmptyCommits, true)
}

func TestDevsInitialize(t *testing.T) {
	d := fixtureDevs()
	assert.NotNil(t, d.days)
}

func TestDevsConsumeFinalize(t *testing.T) {
	devs := fixtureDevs()
	deps := map[string]interface{}{}

	// stage 1
	deps[identity.DependencyAuthor] = 0
	deps[plumbing.DependencyDay] = 0
	cache := map[gitplumbing.Hash]*plumbing.CachedBlob{}
	AddHash(t, cache, "291286b4ac41952cbd1389fda66420ec03c1a9fe")
	AddHash(t, cache, "c29112dbd697ad9b401333b80c18a63951bc18d9")
	AddHash(t, cache, "baa64828831d174f40140e4b3cfa77d1e917a2c1")
	AddHash(t, cache, "dc248ba2b22048cc730c571a748e8ffcf7085ab9")
	deps[plumbing.DependencyBlobCache] = cache
	deps[plumbing.DependencyLanguages] = map[gitplumbing.Hash]string{
		gitplumbing.NewHash("291286b4ac41952cbd1389fda66420ec03c1a9fe"): "Go",
		gitplumbing.NewHash("c29112dbd697ad9b401333b80c18a63951bc18d9"): "Go",
		gitplumbing.NewHash("baa64828831d174f40140e4b3cfa77d1e917a2c1"): "Go",
		gitplumbing.NewHash("dc248ba2b22048cc730c571a748e8ffcf7085ab9"): "Go",
	}
	changes := make(object.Changes, 3)
	treeFrom, _ := test.Repository.TreeObject(gitplumbing.NewHash(
		"a1eb2ea76eb7f9bfbde9b243861474421000eb96"))
	treeTo, _ := test.Repository.TreeObject(gitplumbing.NewHash(
		"994eac1cd07235bb9815e547a75c84265dea00f5"))
	changes[0] = &object.Change{From: object.ChangeEntry{
		Name: "analyser.go",
		Tree: treeFrom,
		TreeEntry: object.TreeEntry{
			Name: "analyser.go",
			Mode: 0100644,
			Hash: gitplumbing.NewHash("dc248ba2b22048cc730c571a748e8ffcf7085ab9"),
		},
	}, To: object.ChangeEntry{
		Name: "analyser.go",
		Tree: treeTo,
		TreeEntry: object.TreeEntry{
			Name: "analyser.go",
			Mode: 0100644,
			Hash: gitplumbing.NewHash("baa64828831d174f40140e4b3cfa77d1e917a2c1"),
		},
	}}
	changes[1] = &object.Change{From: object.ChangeEntry{}, To: object.ChangeEntry{
		Name: "cmd/hercules/main.go",
		Tree: treeTo,
		TreeEntry: object.TreeEntry{
			Name: "cmd/hercules/main.go",
			Mode: 0100644,
			Hash: gitplumbing.NewHash("c29112dbd697ad9b401333b80c18a63951bc18d9"),
		},
	},
	}
	changes[2] = &object.Change{From: object.ChangeEntry{}, To: object.ChangeEntry{
		Name: ".travis.yml",
		Tree: treeTo,
		TreeEntry: object.TreeEntry{
			Name: ".travis.yml",
			Mode: 0100644,
			Hash: gitplumbing.NewHash("291286b4ac41952cbd1389fda66420ec03c1a9fe"),
		},
	},
	}
	deps[plumbing.DependencyTreeChanges] = changes
	fd := fixtures.FileDiff()
	result, err := fd.Consume(deps)
	assert.Nil(t, err)
	deps[plumbing.DependencyFileDiff] = result[plumbing.DependencyFileDiff]
	deps[core.DependencyCommit], _ = test.Repository.CommitObject(gitplumbing.NewHash(
		"cce947b98a050c6d356bc6ba95030254914027b1"))
	deps[core.DependencyIsMerge] = false
	result, err = devs.Consume(deps)
	assert.Nil(t, result)
	assert.Nil(t, err)
	assert.Len(t, devs.days, 1)
	day := devs.days[0]
	assert.Len(t, day, 1)
	dev := day[0]
	assert.Equal(t, dev.Commits, 1)
	assert.Equal(t, dev.Added, 847)
	assert.Equal(t, dev.Removed, 9)
	assert.Equal(t, dev.Changed, 67)
	assert.Equal(t, dev.Languages["Go"].Added, 847)
	assert.Equal(t, dev.Languages["Go"].Removed, 9)
	assert.Equal(t, dev.Languages["Go"].Changed, 67)

	deps[identity.DependencyAuthor] = 1
	result, err = devs.Consume(deps)
	assert.Nil(t, result)
	assert.Nil(t, err)
	assert.Len(t, devs.days, 1)
	day = devs.days[0]
	assert.Len(t, day, 2)
	for i := 0; i < 2; i++ {
		dev = day[i]
		assert.Equal(t, dev.Commits, 1)
		assert.Equal(t, dev.Added, 847)
		assert.Equal(t, dev.Removed, 9)
		assert.Equal(t, dev.Changed, 67)
		assert.Equal(t, dev.Languages["Go"].Added, 847)
		assert.Equal(t, dev.Languages["Go"].Removed, 9)
		assert.Equal(t, dev.Languages["Go"].Changed, 67)
	}

	result, err = devs.Consume(deps)
	assert.Nil(t, result)
	assert.Nil(t, err)
	assert.Len(t, devs.days, 1)
	day = devs.days[0]
	assert.Len(t, day, 2)
	dev = day[0]
	assert.Equal(t, dev.Commits, 1)
	assert.Equal(t, dev.Added, 847)
	assert.Equal(t, dev.Removed, 9)
	assert.Equal(t, dev.Changed, 67)
	assert.Equal(t, dev.Languages["Go"].Added, 847)
	assert.Equal(t, dev.Languages["Go"].Removed, 9)
	assert.Equal(t, dev.Languages["Go"].Changed, 67)
	dev = day[1]
	assert.Equal(t, dev.Commits, 2)
	assert.Equal(t, dev.Added, 847*2)
	assert.Equal(t, dev.Removed, 9*2)
	assert.Equal(t, dev.Changed, 67*2)
	assert.Equal(t, dev.Languages["Go"].Added, 847*2)
	assert.Equal(t, dev.Languages["Go"].Removed, 9*2)
	assert.Equal(t, dev.Languages["Go"].Changed, 67*2)

	deps[plumbing.DependencyDay] = 1
	result, err = devs.Consume(deps)
	assert.Nil(t, result)
	assert.Nil(t, err)
	assert.Len(t, devs.days, 2)
	day = devs.days[0]
	assert.Len(t, day, 2)
	dev = day[0]
	assert.Equal(t, dev.Commits, 1)
	assert.Equal(t, dev.Added, 847)
	assert.Equal(t, dev.Removed, 9)
	assert.Equal(t, dev.Changed, 67)
	assert.Equal(t, dev.Languages["Go"].Added, 847)
	assert.Equal(t, dev.Languages["Go"].Removed, 9)
	assert.Equal(t, dev.Languages["Go"].Changed, 67)
	dev = day[1]
	assert.Equal(t, dev.Commits, 2)
	assert.Equal(t, dev.Added, 847*2)
	assert.Equal(t, dev.Removed, 9*2)
	assert.Equal(t, dev.Changed, 67*2)
	assert.Equal(t, dev.Languages["Go"].Added, 847*2)
	assert.Equal(t, dev.Languages["Go"].Removed, 9*2)
	assert.Equal(t, dev.Languages["Go"].Changed, 67*2)
	day = devs.days[1]
	assert.Len(t, day, 1)
	dev = day[1]
	assert.Equal(t, dev.Commits, 1)
	assert.Equal(t, dev.Added, 847)
	assert.Equal(t, dev.Removed, 9)
	assert.Equal(t, dev.Changed, 67)
	assert.Equal(t, dev.Languages["Go"].Added, 847)
	assert.Equal(t, dev.Languages["Go"].Removed, 9)
	assert.Equal(t, dev.Languages["Go"].Changed, 67)
}

func TestDevsFinalize(t *testing.T) {
	devs := fixtureDevs()
	devs.days[1] = map[int]*DevDay{}
	devs.days[1][1] = &DevDay{10, LineStats{20, 30, 40}, nil}
	x := devs.Finalize().(DevsResult)
	assert.Equal(t, x.Days, devs.days)
	assert.Equal(t, x.reversedPeopleDict, devs.reversedPeopleDict)
}

func TestDevsFork(t *testing.T) {
	devs := fixtureDevs()
	clone := devs.Fork(1)[0].(*DevsAnalysis)
	assert.True(t, devs == clone)
}

func TestDevsSerialize(t *testing.T) {
	devs := fixtureDevs()
	devs.days[1] = map[int]*DevDay{}
	devs.days[1][0] = &DevDay{10, LineStats{20, 30, 40}, map[string]LineStats{"Go": {2, 3, 4}}}
	devs.days[1][1] = &DevDay{1, LineStats{2, 3, 4}, map[string]LineStats{"Go": {25, 35, 45}}}
	devs.days[10] = map[int]*DevDay{}
	devs.days[10][0] = &DevDay{11, LineStats{21, 31, 41}, map[string]LineStats{"": {12, 13, 14}}}
	devs.days[10][identity.AuthorMissing] = &DevDay{
		100, LineStats{200, 300, 400}, map[string]LineStats{"Go": {32, 33, 34}}}
	res := devs.Finalize().(DevsResult)
	buffer := &bytes.Buffer{}
	err := devs.Serialize(res, false, buffer)
	assert.Nil(t, err)
	assert.Equal(t, `  days:
    1:
      0: [10, 20, 30, 40, {Go: [2, 3, 4]}]
      1: [1, 2, 3, 4, {Go: [25, 35, 45]}]
    10:
      0: [11, 21, 31, 41, {none: [12, 13, 14]}]
      -1: [100, 200, 300, 400, {Go: [32, 33, 34]}]
  people:
  - "one@srcd"
  - "two@srcd"
`, buffer.String())

	buffer = &bytes.Buffer{}
	err = devs.Serialize(res, true, buffer)
	assert.Nil(t, err)
	msg := pb.DevsAnalysisResults{}
	proto.Unmarshal(buffer.Bytes(), &msg)
	assert.Equal(t, msg.DevIndex, devs.reversedPeopleDict)
	assert.Len(t, msg.Days, 2)
	assert.Len(t, msg.Days[1].Devs, 2)
	assert.Equal(t, msg.Days[1].Devs[0], &pb.DevDay{
		Commits: 10, Stats: &pb.LineStats{Added: 20, Removed: 30, Changed: 40},
		Languages: map[string]*pb.LineStats{"Go": {Added: 2, Removed: 3, Changed: 4}}})
	assert.Equal(t, msg.Days[1].Devs[1], &pb.DevDay{
		Commits: 1, Stats: &pb.LineStats{Added: 2, Removed: 3, Changed: 4},
		Languages: map[string]*pb.LineStats{"Go": {Added: 25, Removed: 35, Changed: 45}}})
	assert.Len(t, msg.Days[10].Devs, 2)
	assert.Equal(t, msg.Days[10].Devs[0], &pb.DevDay{
		Commits: 11, Stats: &pb.LineStats{Added: 21, Removed: 31, Changed: 41},
		Languages: map[string]*pb.LineStats{"": {Added: 12, Removed: 13, Changed: 14}}})
	assert.Equal(t, msg.Days[10].Devs[-1], &pb.DevDay{
		Commits: 100, Stats: &pb.LineStats{Added: 200, Removed: 300, Changed: 400},
		Languages: map[string]*pb.LineStats{"Go": {Added: 32, Removed: 33, Changed: 34}}})
}

func TestDevsDeserialize(t *testing.T) {
	devs := fixtureDevs()
	devs.days[1] = map[int]*DevDay{}
	devs.days[1][0] = &DevDay{10, LineStats{20, 30, 40}, map[string]LineStats{"Go": {12, 13, 14}}}
	devs.days[1][1] = &DevDay{1, LineStats{2, 3, 4}, map[string]LineStats{"Go": {22, 23, 24}}}
	devs.days[10] = map[int]*DevDay{}
	devs.days[10][0] = &DevDay{11, LineStats{21, 31, 41}, map[string]LineStats{"Go": {32, 33, 34}}}
	devs.days[10][identity.AuthorMissing] = &DevDay{
		100, LineStats{200, 300, 400}, map[string]LineStats{"Go": {42, 43, 44}}}
	res := devs.Finalize().(DevsResult)
	buffer := &bytes.Buffer{}
	err := devs.Serialize(res, true, buffer)
	assert.Nil(t, err)
	rawres2, err := devs.Deserialize(buffer.Bytes())
	assert.Nil(t, err)
	res2 := rawres2.(DevsResult)
	assert.Equal(t, res, res2)
}

func TestDevsMergeResults(t *testing.T) {
	people1 := [...]string{"1@srcd", "2@srcd"}
	people2 := [...]string{"3@srcd", "1@srcd"}
	r1 := DevsResult{
		Days:               map[int]map[int]*DevDay{},
		reversedPeopleDict: people1[:],
	}
	r1.Days[1] = map[int]*DevDay{}
	r1.Days[1][0] = &DevDay{10, LineStats{20, 30, 40}, map[string]LineStats{"Go": {12, 13, 14}}}
	r1.Days[1][1] = &DevDay{1, LineStats{2, 3, 4}, map[string]LineStats{"Go": {22, 23, 24}}}
	r1.Days[10] = map[int]*DevDay{}
	r1.Days[10][0] = &DevDay{11, LineStats{21, 31, 41}, nil}
	r1.Days[10][identity.AuthorMissing] = &DevDay{
		100, LineStats{200, 300, 400}, map[string]LineStats{"Go": {32, 33, 34}}}
	r1.Days[11] = map[int]*DevDay{}
	r1.Days[11][1] = &DevDay{10, LineStats{20, 30, 40}, map[string]LineStats{"Go": {42, 43, 44}}}
	r2 := DevsResult{
		Days:               map[int]map[int]*DevDay{},
		reversedPeopleDict: people2[:],
	}
	r2.Days[1] = map[int]*DevDay{}
	r2.Days[1][0] = &DevDay{10, LineStats{20, 30, 40}, map[string]LineStats{"Go": {12, 13, 14}}}
	r2.Days[1][1] = &DevDay{1, LineStats{2, 3, 4}, map[string]LineStats{"Go": {22, 23, 24}}}
	r2.Days[2] = map[int]*DevDay{}
	r2.Days[2][0] = &DevDay{11, LineStats{21, 31, 41}, map[string]LineStats{"Go": {32, 33, 34}}}
	r2.Days[2][identity.AuthorMissing] = &DevDay{
		100, LineStats{200, 300, 400}, map[string]LineStats{"Go": {42, 43, 44}}}
	r2.Days[10] = map[int]*DevDay{}
	r2.Days[10][0] = &DevDay{11, LineStats{21, 31, 41}, map[string]LineStats{"Go": {52, 53, 54}}}
	r2.Days[10][identity.AuthorMissing] = &DevDay{
		100, LineStats{200, 300, 400}, map[string]LineStats{"Go": {62, 63, 64}}}

	devs := fixtureDevs()
	rm := devs.MergeResults(r1, r2, nil, nil).(DevsResult)
	peoplerm := [...]string{"1@srcd", "2@srcd", "3@srcd"}
	assert.Equal(t, rm.reversedPeopleDict, peoplerm[:])
	assert.Len(t, rm.Days, 4)
	assert.Equal(t, rm.Days[11], map[int]*DevDay{
		1: {10, LineStats{20, 30, 40}, map[string]LineStats{"Go": {42, 43, 44}}}})
	assert.Equal(t, rm.Days[2], map[int]*DevDay{
		identity.AuthorMissing: {100, LineStats{200, 300, 400}, map[string]LineStats{"Go": {42, 43, 44}}},
		2:                      {11, LineStats{21, 31, 41}, map[string]LineStats{"Go": {32, 33, 34}}},
	})
	assert.Equal(t, rm.Days[1], map[int]*DevDay{
		0: {11, LineStats{22, 33, 44}, map[string]LineStats{"Go": {34, 36, 38}}},
		1: {1, LineStats{2, 3, 4}, map[string]LineStats{"Go": {22, 23, 24}}},
		2: {10, LineStats{20, 30, 40}, map[string]LineStats{"Go": {12, 13, 14}}},
	})
	assert.Equal(t, rm.Days[10], map[int]*DevDay{
		0: {11, LineStats{21, 31, 41}, map[string]LineStats{}},
		2: {11, LineStats{21, 31, 41}, map[string]LineStats{"Go": {52, 53, 54}}},
		identity.AuthorMissing: {
			100 * 2, LineStats{200 * 2, 300 * 2, 400 * 2}, map[string]LineStats{"Go": {94, 96, 98}}},
	})
}
