export function fuzzyMatch(query: string, target: string): number {
	const q = query.toLowerCase();
	const t = target.toLowerCase();
	const qLen = q.length;
	const tLen = t.length;

	if (qLen === 0) return 0;
	if (qLen > tLen) return -1;

	let score = 0;
	let qi = 0;
	let consecutiveBonus = 0;
	let prevMatch = -2;

	for (let ti = 0; ti < tLen && qi < qLen; ti++) {
		if (q[qi] === t[ti]) {
			if (ti === prevMatch + 1) {
				consecutiveBonus += 2;
				score += 15 + consecutiveBonus;
			} else if (
				ti === 0 ||
				t[ti - 1] === " " ||
				t[ti - 1] === "-" ||
				t[ti - 1] === "_"
			) {
				consecutiveBonus = 0;
				score += 10;
			} else {
				consecutiveBonus = 0;
				const gap = ti - prevMatch;
				score += Math.max(1, 5 - gap);
			}
			prevMatch = ti;
			qi++;
		}
	}

	if (qi < qLen) return -1;

	score += tLen - qLen <= 3 ? 5 : 0;
	return score;
}

export function fuzzyFilter<T>(
	query: string,
	items: T[],
	getStrings: (item: T) => string[],
): T[] {
	const scored = items
		.map((item) => {
			const strings = getStrings(item);
			const best = Math.max(...strings.map((s) => fuzzyMatch(query, s)));
			return { item, score: best };
		})
		.filter(({ score }) => score > 0);

	scored.sort((a, b) => b.score - a.score);
	return scored.map(({ item }) => item);
}
