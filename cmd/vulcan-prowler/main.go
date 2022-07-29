/*
Copyright 2020 Adevinta
*/

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"

	check "github.com/adevinta/vulcan-check-sdk"
	"github.com/adevinta/vulcan-check-sdk/helpers"
	checkstate "github.com/adevinta/vulcan-check-sdk/state"
	report "github.com/adevinta/vulcan-report"
)

const (
	// defaultAPIRegion defines the default AWS region to use when querying AWS
	// services API endpoints.
	defaultAPIRegion       = `eu-west-1`
	defaultSessionDuration = 3600 // 1 hour.

	envEndpoint = `VULCAN_ASSUME_ROLE_ENDPOINT`
	envRole     = `ROLE_NAME`

	envKeyID     = `AWS_ACCESS_KEY_ID`
	envKeySecret = `AWS_SECRET_ACCESS_KEY`
	envToken     = `AWS_SESSION_TOKEN`
)

var (
	checkName = "vulcan-prowler"
	logger    = check.NewCheckLog(checkName)

	defaultGroups = []string{
		"cislevel2",
	}

	// CISCompliance is the vulnerability generated by the check when it does
	// not receive any security level and the account has failed controls.
	CISCompliance = report.Vulnerability{
		Summary: "Compliance With CIS AWS Foundations Benchmark (BETA)",
		Description: `<p>
			The check did not receive the security classification
			of the AWS account so the benchmark has been executed against the
			CIS Level 2.
			The CIS AWS Foundations Benchmark provides prescriptive
			guidance for configuring security options for a subset of Amazon Web
			Services with an emphasis on foundational, testable, and architecture
			agnostic settings. The services included in the scope are: IAM, Confing,
			CloudTrail, CloudWatch, SNS, S3 and VPC (Default).
		</p>
		<p>
			Recommendations are provided in order to comply with all the controls required
			by the CIS Level 2.
		</p>
		<p>
			Check the Details and Resources sections to know the compliance status
			and more details.
		</p>`,
		Labels: []string{"compliance", "cis", "aws"},
		References: []string{
			"https://d0.awsstatic.com/whitepapers/compliance/AWS_CIS_Foundations_Benchmark.pdf",
			"https://github.com/toniblyx/prowler",
			"https://www.cisecurity.org/benchmark/amazon_web_services/",
		},
		Fingerprint: helpers.ComputeFingerprint(),
		Score:       report.SeverityThresholdMedium,
	}

	// CISLevel1Compliance is the vulnerability generated by the check when it
	// receives a security level of 1 or less and the account has failed
	// controls.
	CISLevel1Compliance = report.Vulnerability{
		Summary: "Compliance With CIS Level 1 AWS Foundations Benchmark (BETA)",
		Description: `<p>
			This account has been checked for compliance with the CIS Level 1 according
			to its security classification. You can check the security classification of the account in
			the details section.
			</p>
			<p>
		    The CIS AWS Foundations Benchmark provides prescriptive
			guidance for configuring security options for a subset of Amazon Web
			Services with an emphasis on foundational, testable, and architecture
			agnostic settings. The services included in the scope are: IAM, Confing,
			CloudTrail, CloudWatch, SNS, S3 and VPC (Default).
		</p>
		<p>
			Recommendations are provided in order to comply with all the controls required
			by the CIS Level 1.
		</p>
		<p>
			Check the Details and Resources sections to know the compliance status
			and more details.
		</p>`,
		Labels: []string{"compliance", "cis", "aws"},
		References: []string{
			"https://d0.awsstatic.com/whitepapers/compliance/AWS_CIS_Foundations_Benchmark.pdf",
			"https://github.com/toniblyx/prowler",
			"https://www.cisecurity.org/benchmark/amazon_web_services/",
		},
		Fingerprint: helpers.ComputeFingerprint(),
		Score:       report.SeverityThresholdMedium,
	}

	// CISLevel2Compliance is the vulnerability generated by the check when it
	// receives a security level of 2 or more and the account has failed
	// controls.
	CISLevel2Compliance = report.Vulnerability{
		Summary: "Compliance With CIS Level 2 AWS Foundations Benchmark (BETA)",
		Description: `<p>
			This account has been checked for compliance with the CIS Level 2 according
			to its security classification. You can check the security classification
			of the account in the details section.
			</p>
			<p>
		    The CIS AWS Foundations Benchmark provides prescriptive
			guidance for configuring security options for a subset of Amazon Web
			Services with an emphasis on foundational, testable, and architecture
			agnostic settings. The services included in the scope are: IAM, Confing,
			CloudTrail, CloudWatch, SNS, S3 and VPC (Default).
		</p>
		<p>
			Recommendations are provided in order to comply with all the controls required
			by the CIS Level 2.
		</p>
		<p>
			Check the Details and Resources sections to know the compliance status
			and more details.
		</p>`,
		Labels: []string{"compliance", "cis", "aws"},
		References: []string{
			"https://d0.awsstatic.com/whitepapers/compliance/AWS_CIS_Foundations_Benchmark.pdf",
			"https://github.com/toniblyx/prowler",
			"https://www.cisecurity.org/benchmark/amazon_web_services/",
		},
		Fingerprint: helpers.ComputeFingerprint(),
		Score:       report.SeverityThresholdMedium,
	}

	// CISComplianceInfo is a vulnerability that is always generated by the
	// check. It contains the not scored and informational controls related to
	// the account.
	CISComplianceInfo = report.Vulnerability{
		Summary: "Information About CIS AWS Foundations Benchmark (BETA)",
		Description: `<p>
			     Information gathered by executing the CIS benchmark on the account.
		</p>
			`,
		Labels: []string{"compliance", "cis", "aws"},
		References: []string{
			"https://d0.awsstatic.com/whitepapers/compliance/AWS_CIS_Foundations_Benchmark.pdf",
			"https://github.com/toniblyx/prowler",
			"https://www.cisecurity.org/benchmark/amazon_web_services/",
		},
		Fingerprint: helpers.ComputeFingerprint(),
		Score:       report.SeverityThresholdNone,
	}
)

// CISControl holds the info related to AWS CIS control.
type CISControl struct {
	ID              string  `json:"id"`
	Severity        float32 `json:"severity"`
	SeverityLiteral string  `json:"severity_literal"`
	Remediation     string  `json:"remediation"`
}

type options struct {
	Region          string   `json:"region"`
	Groups          []string `json:"groups"`
	SessionDuration int      `json:"session_duration"` // In secs.
	SecurityLevel   *byte    `json:"security_level"`
}

func buildOptions(optJSON string) (options, error) {
	var opts options
	if optJSON != "" {
		if err := json.Unmarshal([]byte(optJSON), &opts); err != nil {
			return opts, err
		}
	}
	if opts.Groups == nil {
		opts.Groups = defaultGroups
	}
	if opts.SessionDuration == 0 {
		opts.SessionDuration = defaultSessionDuration
	}

	return opts, nil
}

func main() {
	run := func(ctx context.Context, target, assetType, optJSON string, state checkstate.State) error {
		if target == "" {
			return errors.New("check target missing")
		}
		parsedARN, err := arn.Parse(target)
		if err != nil {
			return err
		}

		opts, err := buildOptions(optJSON)
		if err != nil {
			return err
		}

		endpoint := os.Getenv(envEndpoint)
		if endpoint == "" {
			return fmt.Errorf("%s env var must have a non-empty value", envEndpoint)
		}
		role := os.Getenv(envRole)

		logger.Infof("using endpoint '%s' and role '%s'", endpoint, role)

		isReachable, err := helpers.IsReachable(target, assetType,
			helpers.NewAWSCreds(endpoint, role))
		if err != nil {
			logger.Warnf("Can not check asset reachability: %v", err)
		}
		if !isReachable {
			return checkstate.ErrAssetUnreachable
		}

		if err := loadCredentials(endpoint, parsedARN.AccountID, role, opts.SessionDuration); err != nil {
			return fmt.Errorf("can not get credentials for the role '%s' from the endpoint '%s': %w", endpoint, role, err)
		}

		alias, err := accountAlias(credentials.NewEnvCredentials())
		if err != nil {
			return fmt.Errorf("can not retrieve account alias: %w", err)
		}

		logger.Infof("account alias: '%s'", alias)
		groups, err := groupsFromOpts(opts)
		if err != nil {
			return err
		}
		// Load AWS CIS controls information.
		content, err := ioutil.ReadFile("cis_controls.json")
		if err != nil {
			return err
		}
		controls := map[string]CISControl{}
		err = json.Unmarshal(content, &controls)
		if err != nil {
			return err
		}
		r, err := runProwler(ctx, opts.Region, groups)
		if err != nil {
			return err
		}

		var v report.Vulnerability
		if opts.SecurityLevel == nil {
			v = CISCompliance

		} else if *opts.SecurityLevel == 0 || *opts.SecurityLevel == 1 {
			v = CISLevel1Compliance
		} else {
			v = CISLevel2Compliance
		}
		fv, err := fillCISLevelVuln(&v, r, alias, opts.SecurityLevel, controls)
		if err != nil {
			return err
		}
		infov, err := buildCISInfoVuln(r, alias, opts.SecurityLevel)
		if err != nil {
			return err
		}
		// if fv == nil it means there were no failed checks so there is no
		// vuln.
		if fv != nil {
			state.AddVulnerabilities(*fv)
		}
		state.AddVulnerabilities(infov)

		return nil
	}

	c := check.NewCheckFromHandler(checkName, run)
	c.RunAndServe()
}

func groupsFromOpts(opts options) ([]string, error) {
	// If the security level is specified then it defines the group to use.
	if opts.SecurityLevel == nil {
		return opts.Groups, nil
	}
	level := *opts.SecurityLevel
	if level < 0 || level > 2 {
		return nil, errors.New("invalid security level value")
	}

	if level == 0 || level == 1 {
		return []string{"cislevel1"}, nil
	}
	return []string{"cislevel2"}, nil

}

func buildCISInfoVuln(r *prowlerReport, alias string, slevel *byte) (report.Vulnerability, error) {
	v := CISComplianceInfo
	var info []entry
	infoTable := report.ResourcesGroup{
		Name: "Info + Not Scored Controls",
		Header: []string{
			"Control",
			"Description",
			"Region",
			"Message",
		},
	}
	for _, e := range r.entries {
		switch e.Status {
		case "Info":
			info = append(info, e)
			control, description, err := parseControl(e.Control)
			if err != nil {
				return report.Vulnerability{}, err
			}
			row := map[string]string{
				"Control":     control,
				"Description": description,
				"Region":      e.Region,
				"Message":     e.Message,
			}
			infoTable.Rows = append(infoTable.Rows, row)
		}
	}
	v.Resources = append(v.Resources, infoTable)

	v.Details = fmt.Sprintf("Account: %s\n", alias)
	if slevel != nil {
		v.Details += fmt.Sprintf("Security Level: %d\n", *slevel)
	}
	v.Details += "\n"
	v.Details += fmt.Sprintf("Info + Not Scored Controls: %d\n", len(info))

	return v, nil
}

func fillCISLevelVuln(v *report.Vulnerability, r *prowlerReport, alias string, slevel *byte, controls map[string]CISControl) (*report.Vulnerability, error) {
	type controlRow struct {
		row     map[string]string
		control string
		score   float32
	}
	var (
		total  int
		rows   []controlRow
		failed []entry
	)
	fcTable := report.ResourcesGroup{
		Name: "Failed Controls",
		Header: []string{
			"Control",
			"Description",
			"CIS Severity",
			"Region",
			"Message",
			"Remediation",
		},
	}

	for _, e := range r.entries {
		switch e.Status {
		case "FAIL":
			failed = append(failed, e)
			control, description, err := parseControl(e.Control)
			if err != nil {
				return nil, err
			}
			cinfo, ok := controls[control]
			if !ok {
				return nil, fmt.Errorf("no information for control %s", control)
			}
			row := map[string]string{
				"Control":      control,
				"Description":  description,
				"CIS Severity": cinfo.SeverityLiteral,
				"Region":       e.Region,
				"Message":      e.Message,
				"Remediation":  fmt.Sprintf("<a href=\"%s\">Reference</a>", cinfo.Remediation),
			}
			c := controlRow{row, control, cinfo.Severity}
			rows = append(rows, c)
			fallthrough
		default:
			total++
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].score == rows[j].score {
			return rows[i].control > rows[j].control
		}
		return rows[i].score > rows[j].score
	})
	for _, r := range rows {
		fcTable.Rows = append(fcTable.Rows, r.row)
	}
	v.Resources = append(v.Resources, fcTable)

	v.Details = fmt.Sprintf("Account: %s\n", alias)
	if slevel != nil {
		v.Details += fmt.Sprintf("Security Level: %d\n", *slevel)
	}
	v.Details += "\n"
	v.Details += fmt.Sprintf("Failed Controls: %d\n", len(failed))
	v.Details += fmt.Sprintf("Total Controls: %d\n", total)
	// This vulnerability only makes sense when there is, at least, one failed check.
	if len(failed) < 1 {
		return nil, nil
	}
	return v, nil
}

func parseControl(raw string) (control string, description string, err error) {
	if raw == "" {
		return "", "", fmt.Errorf("error parsing raw control, unexpected format %s", raw)
	}
	// Raw format example: "[check13] Ensure credentials unused for 90 days or
	// greater are disabled (Scored)""
	parts := strings.Split(raw, "] ")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("error parsing raw control, unexpected format %s", raw)
	}
	// parts[0] = [check13 .
	control = strings.Replace(parts[0], "[check", "", -1)
	// control = 13 .
	control = control[:1] + "." + control[1:]
	// control = 1.13
	// description = Ensure credentials unused for 90 days or greater are
	// disabled (Scored)
	description = strings.Replace(parts[1], "(Scored)", "", -1)
	return
}

type assumeRoleResponse struct {
	AccessKey       string `json:"access_key"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token"`
}

func loadCredentials(url string, accountID, role string, sessionDuration int) error {
	m := map[string]interface{}{"account_id": accountID}
	if role != "" {
		m["role"] = role
	}
	if sessionDuration != 0 {
		m["duration"] = sessionDuration
	}
	jsonBody, err := json.Marshal(m)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var r assumeRoleResponse
	err = json.Unmarshal(buf, &r)
	if err != nil {
		logger.Errorf("can not decode response body '%s'", string(buf))
		return err
	}

	if err := os.Setenv(envKeyID, r.AccessKey); err != nil {
		return err
	}

	if err := os.Setenv(envKeySecret, r.SecretAccessKey); err != nil {
		return err
	}

	if err := os.Setenv(envToken, r.SessionToken); err != nil {
		return err
	}

	return nil
}

// accountAlias gets one of the current aliases for the account that the
// credentials passed belong to.
func accountAlias(creds *credentials.Credentials) (string, error) {
	svc := iam.New(session.New(&aws.Config{Credentials: creds}))
	resp, err := svc.ListAccountAliases(&iam.ListAccountAliasesInput{})
	if err != nil {
		return "", err
	}
	if len(resp.AccountAliases) == 0 {
		logger.Warn("No aliases found for the account")
		return "", nil
	}
	a := resp.AccountAliases[0]
	if a == nil {
		return "", errors.New("unexpected nil getting aliases for aws account")
	}
	return *a, nil
}
