package dto

type GlobalStats struct {
	TotalBoards   int
	TotalThreads  int
	TotalPosts    int
	PostsLastHour int
}

type BoardStat struct {
	Slug            string
	Title           string
	ThreadCount     int
	PostCount       int
	PostsLastHour   int
}

type HomePage struct {
	Base
	Global GlobalStats
	Boards []BoardStat
}
