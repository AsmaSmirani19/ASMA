package server

import(

	"database/sql"
	//"fmt"
	//"log"
	//"context"
	//"time"
	//"errors"
	
	//"github.com/lib/pq"

)

func GetPlannedTestInfoByID(db *sql.DB, id int) (*PlannedTestInfo, error) {
    query := `
        SELECT
            t."Id",
            t.test_name,
            t.test_duration,
            t.creation_date,
            sa."Name" AS source_agent,
            ta."Name" AS target_agent,
            th."Name" AS threshold_name,
            CASE
                WHEN th."avg_status" THEN th."avg"
                WHEN th."min_status" THEN th."min"
                WHEN th."max_status" THEN th."max"
                ELSE NULL
            END AS threshold_value,
            p."profile_name" AS profile_name
        FROM test t
        LEFT JOIN "Agent_List" sa ON t.source_id = sa.id
        LEFT JOIN "Agent_List" ta ON t.target_id = ta.id
        LEFT JOIN "Threshold" th ON th."ID" = t.threshold_id
        LEFT JOIN "test_profile" p ON t.profile_id = p."ID"
        WHERE t."Id" = $1
    `

    var info PlannedTestInfo
    row := db.QueryRow(query, id)
    err := row.Scan(
        &info.TestID,
        &info.TestName,
        &info.TestDuration,
        &info.CreationDate,
        &info.SourceAgent,
        &info.TargetAgent,
        &info.ThresholdName,
        &info.ThresholdValue,
        &info.ProfileName,
    )
    if err != nil {
        return nil, err
    }
    return &info, nil
}
