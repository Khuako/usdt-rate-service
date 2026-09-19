create table rates(
    id bigint generated always as identity primary key,
    ask numeric not null,
    bid numeric not null,
    received_at timestamptz not null,
    method text not null constraint method_types check ( method = 'topN' or method = 'avgNM' ),
    n integer not null constraint minimum_n check (n >=1),
    m integer constraint m_constraint check (method = 'topN' and m is null or method = 'avgNM' and m is not null and m >= n)
);