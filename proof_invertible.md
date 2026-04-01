# Proof of MDS Property for Reed-Solomon Encoding Matrices

## 1. Problem Statement

In a systematic Reed-Solomon erasure code with $n$ data shards and $m$ parity shards,
the encoding matrix $E$ is an $(n+m) \times n$ matrix that must satisfy:

1. **Systematic property**: the first $n$ rows form the identity matrix $I_n$.
2. **MDS property**: every $n \times n$ submatrix formed by choosing any $n$ rows of $E$ is invertible.

Property 2 guarantees that the original data can be recovered from any $n$ out of $n+m$ shards,
tolerating up to $m$ lost shards.

All arithmetic is over the finite field $\text{GF}(2^w)$.
The proofs below hold for any $w \ge 1$
(the most common choice in practice is $w = 8$, i.e. $\text{GF}(256)$).

## 2. Two Construction Methods

### 2.1 Standard Method

1. Build an $(n+m) \times n$ matrix $A$ whose every set of $n$ rows is linearly independent
   (e.g. a Vandermonde matrix evaluated at $n+m$ distinct points).
2. Apply Gaussian elimination to transform the first $n$ rows into $I_n$.
3. Elementary row operations preserve the property that every $n$ rows are linearly independent,
   so the resulting matrix satisfies the MDS property.

### 2.2 Lazy Construction

1. Set the first $n$ rows to $I_n$ directly.
2. Set the remaining $m$ rows to some $m \times n$ matrix $G$.

This skips Gaussian elimination entirely.
**Question: does the resulting matrix still satisfy the MDS property?**

The answer depends on the choice of $G$.

## 3. Key Reduction

### 3.1 Block Analysis

Delete any $m$ rows from $E$ to obtain an $n \times n$ square matrix $M$.
Among the $n$ remaining rows:

- $n - a$ rows come from $I_n$; let $S \subset \lbrace 0, \ldots, n-1 \rbrace$ be their row-index set.
- $a$ rows come from $G$ (where $0 \le a \le m$).

Define $T = \lbrace 0, \ldots, n-1 \rbrace \setminus S$, so $|T| = a$.

### 3.2 Reduction via Row Elimination

The rows from $I_n$ indexed by $S$ form an identity submatrix on column set $S$.
Using these rows to perform elementary row operations on the $a$ rows from $G$,
we can eliminate all entries of $G$ in column set $S$.

After elimination, the determinant of $M$ is nonzero if and only if
the $a \times a$ submatrix of $G$ restricted to the selected $a$ rows and column set $T$ is nonzero:

$$
\det(M) \neq 0 \iff \det(G_{\text{rows}, T}) \neq 0
$$

### 3.3 Equivalent Condition

> **The MDS property holds if and only if every square submatrix of $G$
> (any $a$ rows, any $a$ columns, $1 \le a \le \min(m, n)$) is invertible.**

## 4. Vandermonde Matrix: Lazy Construction Is Unsafe

### 4.1 Structure

A Vandermonde matrix $G$ with distinct nonzero parameters
$x_1, \ldots, x_m \in \text{GF}(2^w)^{*}$:

$$
G_{ij} = x_i^{\ j}, \quad 1 \le i \le m,\ 0 \le j \le n-1
$$

Note that the first entry in every row is $x_i^0 = 1$.

### 4.2 Counterexample

Pick any two rows $i_1, i_2$ and column set $\lbrace 0, d \rbrace$.
The corresponding $2 \times 2$ submatrix is:

$$
\begin{pmatrix} 1 & x_{i_1}^{d} \\\\ 1 & x_{i_2}^{d} \end{pmatrix}
$$

Its determinant is $x_{i_2}^{d} + x_{i_1}^{d}$
(addition and subtraction are the same in $\text{GF}(2^w)$).
If $(x_{i_2} \cdot x_{i_1}^{-1})^{d} = 1$, the determinant is zero.

**Concrete counterexample in $\text{GF}(2^8)$:**

The multiplicative group $\text{GF}(2^8)^{*}$ has order $255 = 3 \times 5 \times 17$.
Let $\alpha$ be a primitive element and set $\beta = \alpha^{85}$.
Then $\beta$ has multiplicative order $255 / \gcd(85, 255) = 255 / 85 = 3$.

Choose $x_1 = \alpha^{k}$ and $x_2 = \alpha^{k+85}$ for any $k$. Then:

$$
\left(\frac{x_2}{x_1}\right)^{3} = \beta^{3} = \alpha^{255} = 1
$$

When $n \ge 4$, taking $d = 3$ and column set $\lbrace 0, 3 \rbrace$ gives a singular submatrix.

> **Generalization**: for any $\text{GF}(2^w)$, whenever $2^w - 1$ has a divisor $d < n$,
> there exist parameter choices that make a submatrix singular.
> For example, $w = 8$ gives $d = 3$.
> Only when $2^w - 1$ is prime (Mersenne prime, e.g. $w = 2, 3, 5, 7, \ldots$)
> does this particular class of counterexamples vanish —
> but the most commonly used $w = 8$ does not satisfy this condition.

### 4.3 Conclusion

Not every square submatrix of a Vandermonde matrix is guaranteed to be invertible.
**Lazy construction with a Vandermonde matrix does not ensure the MDS property.**

## 5. Cauchy Matrix: Lazy Construction Is Safe

### 5.1 Definition

Choose two sets of parameters from $\text{GF}(2^w)$:

- $x_1, \ldots, x_m$: pairwise distinct
- $y_1, \ldots, y_n$: pairwise distinct
- $\lbrace x_i \rbrace \cap \lbrace y_j \rbrace = \emptyset$ (equivalently, $x_i + y_j \neq 0$ for all $i, j$)

The Cauchy matrix is defined as:

$$
G_{ij} = \frac{1}{x_i + y_j}, \quad 1 \le i \le m,\ 1 \le j \le n
$$

> **Note**: in $\text{GF}(2^w)$, $x - y = x + y$,
> so this coincides with the classical Cauchy matrix $\frac{1}{x_i - y_j}$.

### 5.2 Cauchy Determinant Formula

For any $a \times a$ Cauchy submatrix
with row set $\lbrace i_1, \ldots, i_a \rbrace$ and column set $\lbrace j_1, \ldots, j_a \rbrace$,
the determinant has the following closed-form expression
(see Schechter 1959, or any standard linear algebra reference):

$$
\det\left(\frac{1}{x_{i_p} + y_{j_q}}\right)_{1 \le p,q \le a} = \frac{\prod_{1 \le p < q \le a}(x_{i_p} + x_{i_q}) \cdot \prod_{1 \le p < q \le a}(y_{j_p} + y_{j_q})}{\prod_{p=1}^{a} \prod_{q=1}^{a}(x_{i_p} + y_{j_q})}
$$

### 5.3 Proof

It suffices to show that the determinant above is nonzero in $\text{GF}(2^w)$.

**Denominator**: every factor $x_{i_p} + y_{j_q} \neq 0$
(because $\lbrace x_i \rbrace \cap \lbrace y_j \rbrace = \emptyset$).
A product of nonzero elements in a finite field is nonzero, so the denominator $\neq 0$.

**Numerator**:

- $\prod_{p < q}(x_{i_p} + x_{i_q})$:
  all $x_i$ are pairwise distinct $\Rightarrow$ every factor is nonzero $\Rightarrow$ the product is nonzero.
- $\prod_{p < q}(y_{j_p} + y_{j_q})$:
  all $y_j$ are pairwise distinct $\Rightarrow$ every factor is nonzero $\Rightarrow$ the product is nonzero.

Numerator nonzero, denominator nonzero, therefore the determinant is nonzero.

Hence **every $a \times a$ square submatrix of $G$ is invertible**. $\blacksquare$

### 5.4 Example Parameter Choice

In $\text{GF}(2^w)$ with primitive element $\alpha$, a simple valid choice is:

$$
x_i = \alpha^{i-1}, \quad i = 1, \ldots, m
$$

$$
y_j = \alpha^{m+j-1}, \quad j = 1, \ldots, n
$$

This uses $m + n$ distinct nonzero field elements
and requires $m + n \le 2^w - 1$
(for $\text{GF}(2^8)$: $m + n \le 255$).

## 6. Summary

| Matrix Type | Lazy Construction Safe? | Reason |
|:-----------:|:-----------------------:|:------:|
| Vandermonde | ❌ | Submatrices can be singular (§4.2 counterexample) |
| Cauchy | ✅ | Every submatrix is invertible (§5.3 determinant formula) |

**Conclusion**: using a Cauchy matrix for the parity rows allows skipping Gaussian elimination
while strictly guaranteeing the MDS property.