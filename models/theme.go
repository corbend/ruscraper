package models

import (
	"fmt"
	"time"
	"encoding/json"
	"gopkg.in/olivere/elastic.v3"
)

type ParseUrl struct {
	Url string `json:"url"`
	IndexName string `json:"index_name"`
}

type Theme struct {
	Id int64
	Read bool
	Name string
	Size string
	Date string
	Answers string
	PubYear int
	CreateDate int64
}

type Page struct {
	Number int
	Themes []Theme
}

type ThemeFinder struct {
	name string
	pub_year int
	create_date int64
	create_date_from int64
	create_date_to int64
	from int
	size int
}

// var compiledQueries = map[string]func(service *elastic.SearchService) *elastic.SearchService{
// 	"LastDay": GetLastDay,
// }

func (f *ThemeFinder) Find(client *elastic.Client, query *elastic.Query) ([]*Theme, error) {

	search := client.Search().Index("hollywood").Type("film")

	if query != nil {
		search := query
	} else {
		search = f.Query(search)
	}

    search = f.Query(search)
    search = f.Aggregate(search)
    // search = f.sorting(search)
    // search = f.paginate(search)

    // TODO Add other properties here, e.g. timeouts, explain or pretty printing

    // Execute query
    sr, err := search.Do()
    if err != nil {
        return nil, err
    }

    return f.Decode(sr)
}

func NewFinder() *ThemeFinder {
    return &ThemeFinder{}
}

// Genre filters the results by the given genre.
func (f *ThemeFinder) Name(name string) *ThemeFinder {
    f.name = name
    return f
}

// Year filters the results by the specified year.
func (f *ThemeFinder) Year(year int) *ThemeFinder {
    f.pub_year = year
    return f
}

func (f *ThemeFinder) CreateDate(create_date int64) *ThemeFinder {
    f.create_date = create_date
    return f
}

func (f *ThemeFinder) CreateDateFrom(create_date_from int64) *ThemeFinder {
    f.create_date_from = create_date_from
    return f
}

func (f *ThemeFinder) CreateDateTo(create_date_to int64) *ThemeFinder {
    f.create_date_to = create_date_to
    return f
}

// From specifies the start index for pagination.
func (f *ThemeFinder) From(from int) *ThemeFinder {
    f.from = from
    return f
}

// Size specifies the number of items to return in pagination.
func (f *ThemeFinder) Size(size int) *ThemeFinder {
    f.size = size
    return f
}

func (f *ThemeFinder) Decode(res *elastic.SearchResult) ([]*Theme, error) {
    if res == nil || res.TotalHits() == 0 {
        return nil, nil
    }

    var recs []*Theme
    for _, hit := range res.Hits.Hits {
        r := new(Theme)
        if err := json.Unmarshal(*hit.Source, r); err != nil {
            return nil, err
        }
        // TODO Add Score here, e.g.:
        // film.Score = *hit.Score
        recs = append(recs, r)
    }
    return recs, nil
}

func (f *ThemeFinder) Query(service *elastic.SearchService) *elastic.SearchService {
    if f.name == "" && f.pub_year == 0 && f.create_date == 0 {
        service = service.Query(elastic.NewMatchAllQuery())
        return service
    }

    q := elastic.NewBoolQuery()
    if f.name != "" {
        q = q.Must(elastic.NewTermQuery("Name", f.Name))
    }
    if f.pub_year > 0 {
        q = q.Must(elastic.NewTermQuery("PubYear", f.pub_year))
    }
    if f.create_date > 0 {
    	q = q.Must(elastic.NewTermQuery("CreateDate", f.create_date))
    }

    // TODO Add other queries and filters here, maybe differentiating between AND/OR etc.

    service = service.Query(q)
    return service
}

func (f *ThemeFinder) Aggregate(service *elastic.SearchService) *elastic.SearchService {

	agg := elastic.NewTermsAggregation().Field("Id")
    service = service.Aggregation("Count", agg)
    return service
}

func (f *ThemeFinder) GetLastDay() elastic.Query {

	qu := elastic.NewRangeQuery("CreateDate").From(f.create_date_from).To(f.create_date_to)
	q, _ := qu.Source()
	return q
}

func GetLastThemes(from int64, to int64) ([]*Theme, error) {
	year, month, day := time.Now().Date()
	nowDayTime, _ := time.Parse(time.RFC3339, fmt.Sprintf("%02d-%02d-%02dT00:00:00+00:00", year, month, day))
	finder := ThemeFinder{}

	finder.CreateDateFrom(nowDayTime.Unix())
	finder.CreateDateTo(time.Now().Unix())

	return finder.Find(core.Units.Elastic, finder.GetLastDay())
}
