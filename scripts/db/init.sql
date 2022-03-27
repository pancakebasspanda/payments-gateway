create type payment_status as enum (
    'APPROVED',
    'PENDING',
    'REJECTED',
    'AUTHORIZED',
    'CARD_VERIFIED'
    );

alter type payment_status owner to user1;

create type payment_type as enum (
    'CARD',
    'MOBILE_WALLET',
    'EFT'
);

alter type payment_type owner to user1;

create table payment_details
(
    ref_id              varchar                             NOT NULL,
    name                varchar                             NOT NULL,
    surname             varchar                             NOT NULL,
    email               varchar,
    phone               varchar,
    address_line_1      varchar,
    address_line_2      varchar,
    postcode            varchar,
    card_number         varchar                             NOT NULL,
    currency            varchar                             NOT NULL,
    amount              REAL NULL,
    payment_type        payment_type NULL,
    status              payment_status NULL,
    status_reason             varchar,
    updated_timestamp timestamp default CURRENT_TIMESTAMP not null,
    insert_timestamp    timestamp default CURRENT_TIMESTAMP not null,
    PRIMARY KEY (ref_id)
);


alter table
    payment_details owner to user1;
