package awsenv

import (

	"io/ioutil"

	"strings"

	"gopkg.in/yaml.v2"
	"github.com/kataras/golog"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"

	"strconv"
)

type SsmParameter struct {
	paramName string
	paramType string
	version	  string
	length    string
	value     string
}

func CreateSSMParameters(paramsFile string, region string) {

    params, err := loadParams(paramsFile)
    if err != nil {
		golog.Fatal(err)
	}

	ssmClient := createSSMClient(region)

	m := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(params), m)
	paramMap := generateParamValues(m)
	for _, value := range paramMap {
		createParam(ssmClient, value)
	}
}

func loadParams(params string) ([]byte, error) {
	golog.Infof("loading ssm params for %s", params)
	return ioutil.ReadFile(params)
}

func createSSMClient(region string) *ssm.SSM {
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
	svc := ssm.New(sess)
	return svc
}

func generateParamValues(thing map[string]interface{}) map[string]*SsmParameter {
	result := Flatten(thing)
	return result
}

func createParam(ssmClient *ssm.SSM, param *SsmParameter) {
	if isParamNewOrUpdated(ssmClient, param) {
		golog.Infof("creating or updating ssm parameter %s", param.paramName)
		input := &ssm.PutParameterInput{
			Name:      aws.String(param.paramName),
			Type:      aws.String(param.paramType),
			Value:     aws.String(param.value),
			Overwrite: aws.Bool(true),
		}
		_, err := ssmClient.PutParameter(input)
		if err != nil {
			golog.Fatalf("failed to create param %s", err)
		}
	} else {
		golog.Infof("ssm parameter %s already exists", param.paramName)
	}
}

func isParamNewOrUpdated(ssmClient *ssm.SSM, ssmParam *SsmParameter) bool {
	params := getParam(ssmClient, ssmParam.paramName)
	switch ssmParam.paramType {
	case "SecureString":
		v, err := strconv.ParseInt(ssmParam.version, 10, 64)
		if err != nil {
			golog.Fatalf("unable to parse version %s", err)
		}
		if len(params) > 0 {
			return aws.Int64Value(params[0].Version) < v
		} else {
			return true
		}
	default:
		return len(params) == 0 || aws.StringValue(params[0].Value) != ssmParam.value
	}
	return false
	//return param == nil || aws.StringValue(param.Value) != ssmParam.value
}

func getParam(ssmClient *ssm.SSM, paramName string) []*ssm.ParameterHistory {
	result, err := ssmClient.GetParameterHistory(&ssm.GetParameterHistoryInput{
		Name: aws.String(paramName),
	})
	if err != nil {
		if strings.Contains(err.Error(), ssm.ErrCodeParameterNotFound) {
			return nil
		}
		golog.Fatalf("failed to get param %s caused by %s", paramName, err)
	}
	return result.Parameters
}

