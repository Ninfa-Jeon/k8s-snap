SELECT
    EXISTS (
        SELECT 1
        FROM helm_charts AS c
        WHERE ( c.name = ? ) AND ( c.version = ? )
    )
