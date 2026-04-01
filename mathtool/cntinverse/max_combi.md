## Finding the Maximum of $\binom{a}{b}$ with Respect to $b$

**Claim.** For a fixed non-negative integer $a$, the binomial coefficient
$\binom{a}{b}$ is maximized when $b = \lfloor a/2 \rfloor$
(or equivalently $b = \lceil a/2 \rceil$).

**Proof.**

### Setup

The binomial coefficient $\binom{a}{b}$ is defined for
$b \in \{0, 1, \ldots, a\}$, which is a finite set.
Therefore, a global maximum must exist. Let $b'$ be a value of $b$
at which this global maximum is attained.

### Key Idea

Since $b'$ is a **global** maximizer, it must in particular be at least
as large as both of its immediate neighbors:

$$
① \quad \binom{a}{b'} \geq \binom{a}{b'-1}, \qquad
② \quad \binom{a}{b'} \geq \binom{a}{b'+1}.
$$

These are **necessary conditions** for $b'$ to be the maximizer.
If they pin down $b'$ to a unique value (or a small set of values
that can be verified), the proof is complete — because we already know
a maximizer exists, and it **must** satisfy these conditions.

### From ①

$$
\frac{a!}{b'!\,(a-b')!} \geq \frac{a!}{(b'-1)!\,(a+1-b')!}
$$

Since the numerators are identical, this is equivalent to requiring
the left denominator to be no larger:

$$
b'!\,(a-b')! \leq (b'-1)!\,(a+1-b')!
$$

Dividing both sides by $(b'-1)!\,(a-b')!$:

$$
b' \leq a + 1 - b' \implies b' \leq \frac{a+1}{2} \tag{1}
$$

### From ②

$$
\frac{a!}{b'!\,(a-b')!} \geq \frac{a!}{(b'+1)!\,(a-1-b')!}
$$

Similarly:

$$
b'!\,(a-b')! \leq (b'+1)!\,(a-1-b')!
$$

Dividing both sides by $b'!\,(a-1-b')!$:

$$
a - b' \leq b' + 1 \implies b' \geq \frac{a-1}{2} \tag{2}
$$

### Combining (1) and (2)

$$
\frac{a-1}{2} \leq b' \leq \frac{a+1}{2}
$$

Since $b'$ must be an integer:

- If $a$ is even: the only integer in
  $\left[\frac{a-1}{2},\;\frac{a+1}{2}\right]$ is
  $b' = \dfrac{a}{2}$ (unique maximizer).
- If $a$ is odd: $b' = \dfrac{a-1}{2}$ or $b' = \dfrac{a+1}{2}$
  (both give the same value by symmetry
  $\binom{a}{k} = \binom{a}{a-k}$).

### Why This Proof Works (Logical Completeness)

One might worry: we only used **necessary** conditions —
how do we know these conditions are also **sufficient**?

The answer is:

1. A global maximizer **must exist** (finite set).
2. Any global maximizer **must satisfy** conditions ① and ②.
3. Conditions ① and ② **pin down** the candidate to essentially
   one value ($\lfloor a/2 \rfloor$).
4. Therefore, this candidate **must be** the global maximizer —
   there is simply no other possibility.

This is analogous to: if you know a solution to an equation exists,
and you find that only one value satisfies the necessary conditions,
then that value must be the solution.

Therefore, $\binom{a}{b}$ is maximized at
$b = \lfloor a/2 \rfloor$. $\blacksquare$