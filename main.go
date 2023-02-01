package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func main() {
	blue := "blue.example.com."
	blueType := "A"
	green := "green.example.com."
	greenType := "A"
	zoneID := "Z*******************"

	sess := session.Must(session.NewSession())
	route53Svc := route53.New(sess)

	blueRecord, _ := route53Svc.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(zoneID),
		StartRecordName: aws.String(blue),
		StartRecordType: aws.String(blueType),
		MaxItems:        aws.String("1"),
	})

	greenRecord, _ := route53Svc.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(zoneID),
		StartRecordName: aws.String(green),
		StartRecordType: aws.String(greenType),
		MaxItems:        aws.String("1"),
	})

	originalRecordSetsJSON, _ := json.Marshal([]*route53.Change{
		{
			Action:            aws.String("UPSERT"),
			ResourceRecordSet: blueRecord.ResourceRecordSets[0],
		},
		{
			Action:            aws.String("UPSERT"),
			ResourceRecordSet: greenRecord.ResourceRecordSets[0],
		},
	})

	blueRecord.ResourceRecordSets[0].Name = aws.String(green)
	greenRecord.ResourceRecordSets[0].Name = aws.String(blue)
	changeRecordSets := []*route53.Change{
		{
			Action:            aws.String("UPSERT"),
			ResourceRecordSet: blueRecord.ResourceRecordSets[0],
		},
		{
			Action:            aws.String("UPSERT"),
			ResourceRecordSet: greenRecord.ResourceRecordSets[0],
		},
	}

	if os.Getenv("DRYRUN") == "1" {
		dmp := diffmatchpatch.New()
		var buf bytes.Buffer
		var buf2 bytes.Buffer
		changeRecordSetsJSON, _ := json.Marshal(changeRecordSets)
		_ = json.Indent(&buf, originalRecordSetsJSON, "", "  ")
		_ = json.Indent(&buf2, changeRecordSetsJSON, "", "  ")
		a, b, c := dmp.DiffLinesToChars(buf.String(), buf2.String())
		diffs := dmp.DiffMain(a, b, false)
		diffs = dmp.DiffCharsToLines(diffs, c)
		fmt.Println(dmp.DiffPrettyText(diffs))
		os.Exit(0)
	}

	_, err := route53Svc.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		ChangeBatch: &route53.ChangeBatch{
			Comment: aws.String("swap"),
			Changes: changeRecordSets,
		},
	})
	if err != nil {
		fmt.Println("Error: ", err)
	}
}
