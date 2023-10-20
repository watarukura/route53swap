package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var blue string
var blueType string
var green string
var greenType string
var zoneID string
var isDryRun bool

func init() {
	flag.StringVar(&blue, "blue", "", "swap target 1 like blue.example.com.")
	flag.StringVar(&blueType, "blueType", "", "swap target type 1 like A")
	flag.StringVar(&green, "green", "", "swap target 2 like green.example.com.")
	flag.StringVar(&greenType, "greenType", "", "swap target type 2 like CNAME")
	flag.StringVar(&zoneID, "zoneID", "", "zone id like ZHOGEHOGE")
	flag.BoolVar(&isDryRun, "dryrun", false, "dryrun and print diff")

	flag.Parse()

	if blue == "" || blueType == "" || green == "" || greenType == "" || zoneID == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	if os.Getenv("DRYRUN") == "1" {
		isDryRun = true
	}
}

func main() {
	sess := session.Must(session.NewSession())
	route53Svc := route53.New(sess)

	blueRecord, err := route53Svc.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(zoneID),
		StartRecordName: aws.String(blue),
		StartRecordType: aws.String(blueType),
		MaxItems:        aws.String("1"),
	})
	if err != nil {
		panic(err)
	}

	greenRecord, err := route53Svc.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(zoneID),
		StartRecordName: aws.String(green),
		StartRecordType: aws.String(greenType),
		MaxItems:        aws.String("1"),
	})
	if err != nil {
		panic(err)
	}

	originalRecordSetsJSON, err := json.Marshal([]*route53.Change{
		{
			Action:            aws.String("UPSERT"),
			ResourceRecordSet: blueRecord.ResourceRecordSets[0],
		},
		{
			Action:            aws.String("UPSERT"),
			ResourceRecordSet: greenRecord.ResourceRecordSets[0],
		},
	})
	if err != nil {
		panic(err)
	}

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

	if isDryRun {
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

	_, err = route53Svc.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
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
