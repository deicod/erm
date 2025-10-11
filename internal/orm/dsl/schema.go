package dsl

import publicdsl "github.com/deicod/erm/orm/dsl"

type (
	Schema             = publicdsl.Schema
	Annotation         = publicdsl.Annotation
	GraphQLOption      = publicdsl.GraphQLOption
	SubscriptionEvent  = publicdsl.SubscriptionEvent
	ComparisonOperator = publicdsl.ComparisonOperator
	SortDirection      = publicdsl.SortDirection
	AggregateFunc      = publicdsl.AggregateFunc
	FieldType          = publicdsl.FieldType
	ExpressionSpec     = publicdsl.ExpressionSpec
	IdentityMode       = publicdsl.IdentityMode
)

const (
	AnnotationGraphQL       = publicdsl.AnnotationGraphQL
	SubscriptionEventCreate = publicdsl.SubscriptionEventCreate
	SubscriptionEventUpdate = publicdsl.SubscriptionEventUpdate
	SubscriptionEventDelete = publicdsl.SubscriptionEventDelete
	OpEqual                 = publicdsl.OpEqual
	OpNotEqual              = publicdsl.OpNotEqual
	OpGreaterThan           = publicdsl.OpGreaterThan
	OpLessThan              = publicdsl.OpLessThan
	OpGTE                   = publicdsl.OpGTE
	OpLTE                   = publicdsl.OpLTE
	OpILike                 = publicdsl.OpILike
	SortAsc                 = publicdsl.SortAsc
	SortDesc                = publicdsl.SortDesc
	AggCount                = publicdsl.AggCount
	AggSum                  = publicdsl.AggSum
	AggAvg                  = publicdsl.AggAvg
	AggMin                  = publicdsl.AggMin
	AggMax                  = publicdsl.AggMax
	TypeUUID                = publicdsl.TypeUUID
	TypeText                = publicdsl.TypeText
	TypeVarChar             = publicdsl.TypeVarChar
	TypeChar                = publicdsl.TypeChar
	TypeBoolean             = publicdsl.TypeBoolean
	TypeSmallInt            = publicdsl.TypeSmallInt
	TypeInteger             = publicdsl.TypeInteger
	TypeBigInt              = publicdsl.TypeBigInt
	TypeSmallSerial         = publicdsl.TypeSmallSerial
	TypeSerial              = publicdsl.TypeSerial
	TypeBigSerial           = publicdsl.TypeBigSerial
	TypeDecimal             = publicdsl.TypeDecimal
	TypeNumeric             = publicdsl.TypeNumeric
	TypeReal                = publicdsl.TypeReal
	TypeDoublePrecision     = publicdsl.TypeDoublePrecision
	TypeMoney               = publicdsl.TypeMoney
	TypeBytea               = publicdsl.TypeBytea
	TypeDate                = publicdsl.TypeDate
	TypeTime                = publicdsl.TypeTime
	TypeTimeTZ              = publicdsl.TypeTimeTZ
	TypeTimestamp           = publicdsl.TypeTimestamp
	TypeTimestampTZ         = publicdsl.TypeTimestampTZ
	TypeInterval            = publicdsl.TypeInterval
	TypeJSON                = publicdsl.TypeJSON
	TypeJSONB               = publicdsl.TypeJSONB
	TypeXML                 = publicdsl.TypeXML
	TypeInet                = publicdsl.TypeInet
	TypeCIDR                = publicdsl.TypeCIDR
	TypeMACAddr             = publicdsl.TypeMACAddr
	TypeMACAddr8            = publicdsl.TypeMACAddr8
	TypeBit                 = publicdsl.TypeBit
	TypeVarBit              = publicdsl.TypeVarBit
	TypeTSVector            = publicdsl.TypeTSVector
	TypeTSQuery             = publicdsl.TypeTSQuery
	TypePoint               = publicdsl.TypePoint
	TypeLine                = publicdsl.TypeLine
	TypeLseg                = publicdsl.TypeLseg
	TypeBox                 = publicdsl.TypeBox
	TypePath                = publicdsl.TypePath
	TypePolygon             = publicdsl.TypePolygon
	TypeCircle              = publicdsl.TypeCircle
	TypeInt4Range           = publicdsl.TypeInt4Range
	TypeInt8Range           = publicdsl.TypeInt8Range
	TypeNumRange            = publicdsl.TypeNumRange
	TypeTSRange             = publicdsl.TypeTSRange
	TypeTSTZRange           = publicdsl.TypeTSTZRange
	TypeDateRange           = publicdsl.TypeDateRange
	TypeArray               = publicdsl.TypeArray
	TypeGeometry            = publicdsl.TypeGeometry
	TypeGeography           = publicdsl.TypeGeography
	TypeVector              = publicdsl.TypeVector
	TypeString              = publicdsl.TypeString
	TypeInt                 = publicdsl.TypeInt
	TypeFloat               = publicdsl.TypeFloat
	TypeBool                = publicdsl.TypeBool
	TypeBytes               = publicdsl.TypeBytes
	IdentityByDefault       = publicdsl.IdentityByDefault
	IdentityAlways          = publicdsl.IdentityAlways
)

func GraphQL(name string, opts ...GraphQLOption) Annotation {
	return publicdsl.GraphQL(name, opts...)
}

func GraphQLSubscriptions(events ...SubscriptionEvent) GraphQLOption {
	return publicdsl.GraphQLSubscriptions(events...)
}

func Expression(sql string, deps ...string) ExpressionSpec {
	return publicdsl.Expression(sql, deps...)
}
