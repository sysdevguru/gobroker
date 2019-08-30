# Portfolio Performance Calc

Calculate time weighted portfolio return.

- Cumulative Profit Loss Ratio (t) =
    - CUMPROD(Profit Loss Ratio (t) + 1) - 1

- Profit Loss Ratio (t) =
    - Profit Loss (t) / Total Equity (t - 1)

- Profit Loss (t)
    - p = raw position
    - Intraday Cost Basis (t) = 
        - Either(p.entry_timestamp < MarketOpen(t), EOD Price(t-1), p.entry_price)
    - End of Day Price (t) = 
        - Either(p.exit_timestamp < MarketClose(t), p.exit_price, EOD Price(t))
    - Trade Profit Loss (t) = 
        - SUM((End of Day Price(t) - Intraday Cost basis (t)) / Intraday Cost Basis (t) * p.qty)
    - Profit Loss(t) = (Profit Loss (t) + Dividends(t) - Fees(t))

- Total Equity (t) =
    - Net Balance(t) + Position Market Value(t)
        - Net Balance(t) = `summary.TotalEquity.Sub(*summary.PositionMarketValue)` 


### Misc

- We need to change the way to calculate total equity if we start leveraged trading or short selling.
