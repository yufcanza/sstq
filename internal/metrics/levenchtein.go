package metrics

type LevenshteinResult struct {
	Distance int
	Matrix   [][]int
	Ops      []string
}

func Levenshtein(a, b []string) LevenshteinResult {
	m, n := len(a), len(b)

	if m == 0 && n == 0 {
		return LevenshteinResult{
			Distance: 0,
			Matrix:   [][]int{{0}},
			Ops:      []string{},
		}
	}
	if m == 0 {
		ops := make([]string, n)
		for i := range ops {
			ops[i] = "I"
		}
		return LevenshteinResult{
			Distance: n,
			Matrix:   make([][]int, 1),
			Ops:      ops,
		}
	}
	if n == 0 {
		ops := make([]string, m)
		for i := range ops {
			ops[i] = "D"
		}
		return LevenshteinResult{
			Distance: m,
			Matrix:   make([][]int, 1),
			Ops:      ops,
		}
	}

	matrix := make([][]int, m+1) //матрица (m+1)x(n+1)
	for i := range matrix {
		matrix[i] = make([]int, n+1)
	}
	for i := 0; i <= m; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= n; j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,
				matrix[i][j-1]+1,
				matrix[i-1][j-1]+cost,
			)
		}
	}
	ops := backtrack(matrix, a, b)

	return LevenshteinResult{
		Distance: matrix[m][n],
		Matrix:   matrix,
		Ops:      ops,
	}

}

func backtrack(matrix [][]int, a, b []string) []string {
	i, j := len(a), len(b)
	ops := make([]string, 0, i+j)

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && matrix[i][j] == matrix[i-1][j-1] && a[i-1] == b[j-1] {
			ops = append([]string{"M"}, ops...)
			i--
			j--
			continue
		}
		if i > 0 && j > 0 && matrix[i][j] == matrix[i-1][j-1]+1 {
			ops = append([]string{"S"}, ops...)
			i--
			j--
			continue
		}
		if i > 0 && matrix[i][j] == matrix[i-1][j]+1 {
			ops = append([]string{"D"}, ops...)
			i--
			continue
		}
		if j > 0 && matrix[i][j] == matrix[i][j-1]+1 {
			ops = append([]string{"I"}, ops...)
			j--
			continue
		}

		if i > 0 {
			ops = append([]string{"D"}, ops...)
			i--
		} else if j > 0 {
			ops = append([]string{"I"}, ops...)
			j--
		}
	}
	return ops
}
